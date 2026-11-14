package engine

import (
	"context"
	"reflect"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func TestCashierCanCloseDuringPaymentAndReopen(t *testing.T) {
	students := make([]agents.Student, 12)
	for i := range students {
		students[i] = mealStudent()
		students[i].ID = i + 1
	}
	game, err := NewGame(neutralMealConfig(), students)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	step := func() {
		t.Helper()
		if _, err := game.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	second := game.engine.staff[environment.Servers+1]
	for second.job == nil && game.engine.now < time.Minute {
		step()
	}
	if second.job == nil {
		t.Fatal("第二个收银窗口应正在付款")
	}
	payingStudent := second.job.studentID
	before := game.Report()
	if err := game.SetCashiers(1); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, game.Report()) || second.job.studentID != payingStudent {
		t.Fatal("关闭收银不能重置时间、统计、库存或取消正在进行的付款")
	}
	view := game.Snapshot().Workers[environment.Servers+1]
	if view.Open || view.StudentID != payingStudent {
		t.Fatal("快照应显示停止接单，同时仍在处理当前学生")
	}
	closedAt, eventCount := game.engine.now, len(game.Report().Events)
	for game.engine.now < closedAt+16*time.Second {
		step()
	}
	if second.job != nil || game.engine.queues[agents.Payment].Capacity() != 1 {
		t.Fatal("当前付款完成后，第二个窗口应关闭，付款区容量应降为一")
	}
	if game.engine.ID2Student[payingStudent].result.Status != agents.Completed {
		t.Fatal("关闭窗口前已开始付款的学生必须正常完成")
	}
	for _, event := range game.Report().Events[eventCount:] {
		if event.Signal == agents.PaymentStarted && event.StaffID == second.result.Staff.ID {
			t.Fatal("已关闭的窗口不能再接单")
		}
	}
	if err := game.SetCashiers(2); err != nil {
		t.Fatal(err)
	}
	eventCount = len(game.Report().Events)
	for !game.Snapshot().Done {
		step()
	}
	reopened := false
	for _, event := range game.Report().Events[eventCount:] {
		if event.Signal == agents.PaymentStarted && event.StaffID == second.result.Staff.ID {
			reopened = true
		}
	}
	if !reopened || game.Report().Metrics.Completed != len(students) {
		t.Fatal("重新开放后应继续接单，所有学生都能完成")
	}
	paid := make(map[int]int)
	for _, event := range game.Report().Events {
		if event.Signal == agents.Paid {
			paid[event.StudentID]++
		}
	}
	for _, student := range students {
		if paid[student.ID] != 1 {
			t.Fatalf("学生 %d 的付款次数应为一，实际为 %d", student.ID, paid[student.ID])
		}
	}
	for _, stock := range game.Report().Stock {
		if stock.Reserved != 0 || stock.Remaining+stock.Taken != stock.Dish.Portions+stock.Replenished {
			t.Fatal("收银切换后库存应守恒，且无未交付的预留")
		}
	}
	if err := game.SetCashiers(1); err == nil {
		t.Fatal("结束后的模拟不能再调整窗口")
	}
}

func TestCashierRejectsInvalidCountsWithoutChangingState(t *testing.T) {
	game, err := NewGame(neutralMealConfig(), []agents.Student{mealStudent()})
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	want := game.Snapshot()
	for _, count := range []int{-1, 0, 3} {
		if err := game.SetCashiers(count); err == nil || !reflect.DeepEqual(want, game.Snapshot()) {
			t.Fatalf("非法窗口数 %d 应被拒绝且不改变模拟", count)
		}
	}
}

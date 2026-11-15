package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func TestGameCompletesThroughBothEntryPoints(t *testing.T) {
	config := environment.DefaultConfig()
	config.Tick = time.Second
	generator := environment.DefaultStudentGenerator(config.Tick)
	students, err := generator.Generate(90)
	if err != nil {
		t.Fatal(err)
	}
	continuousReport, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	game, err := NewGame(config, students)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	for {
		done, err := game.Step(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
	}
	// 每轮重新随机抽取选菜与耗时，两种入口都应使所有学生进入结束状态。
	for _, report := range []Report{continuousReport, game.Report()} {
		metrics := report.Metrics
		if metrics.Completed+metrics.Abandoned+metrics.Cancelled != len(students) {
			t.Fatal("连续运行与逐步推进都必须处理完所有学生")
		}
		for _, student := range report.Students {
			if student.Status == agents.Active || student.Status == agents.Scheduled {
				t.Fatal("结束后不能残留正在服务或尚未入场的学生")
			}
		}
	}
	if !game.Snapshot().Done {
		t.Fatal("完成后必须标记结束")
	}
}

func TestSnapshotIsIndependentAndReadingDoesNotAdvanceTime(t *testing.T) {
	config := environment.DefaultConfig()
	config.Tick = time.Second
	student := agents.Student{ID: 1, Preference: agents.Vegetarian, Satisfaction: 50, Patience: time.Hour, StartersWanted: 1, DessertsWanted: 1}
	game, err := NewGame(config, []agents.Student{student})
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	for game.Snapshot().Students[0].Meal.Main == nil && game.engine.now < environment.MaxSelfServiceTime+environment.MaxMainServiceTime+2*config.Tick {
		if _, err := game.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	want := game.Snapshot()
	changed := game.Snapshot()
	if changed.Students[0].Meal.Main == nil {
		t.Fatal("本场景应已交付主菜")
	}
	changed.Students[0].Meal.Main.Name = "被外部修改"
	changed.Students[0].Meal.Starters[0].Name = "被外部修改"
	changed.Students[0].Waiting[agents.Inside] = time.Hour
	changed.Students[0].Path[0] = agents.Outside
	changed.Stock[0].Remaining = -100
	changed.PeakOccupancy[agents.Inside] = 999
	changed.Metrics.AverageWaitByArea[agents.Inside] = time.Hour
	for i := range changed.Zones {
		if len(changed.Zones[i].StudentIDs) > 0 {
			changed.Zones[i].StudentIDs[0] = 999
		}
	}
	for i := range changed.Events {
		if len(changed.Events[i].DishIDs) > 0 {
			changed.Events[i].DishIDs[0] = "被外部修改"
		}
	}
	if !reflect.DeepEqual(want, game.Snapshot()) {
		t.Fatal("读取或修改快照不应推进时间或改变模拟内部状态")
	}
}

func TestGameCloseReturnsReservationAndIsIdempotent(t *testing.T) {
	config := environment.DefaultConfig()
	config.Tick = time.Second
	student := agents.Student{ID: 1, Preference: agents.Vegetarian, Satisfaction: 50, Patience: time.Hour}
	game, err := NewGame(config, []agents.Student{student})
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	reserved := 0
	for reserved == 0 && game.engine.now < environment.MaxSelfServiceTime+2*config.Tick {
		if _, err := game.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		for _, stock := range game.Snapshot().Stock {
			reserved += stock.Reserved
		}
	}
	if reserved != 1 {
		t.Fatal("关闭前应有一份预留主菜")
	}
	game.Close()
	game.Close()
	for _, stock := range game.Snapshot().Stock {
		if stock.Reserved != 0 || stock.Remaining+stock.Taken != stock.Dish.Portions+stock.Replenished {
			t.Fatal("关闭窗口必须释放预留库存")
		}
	}
	if done, err := game.Step(context.Background()); !done || !errors.Is(err, context.Canceled) {
		t.Fatal("已关闭的会话不得再次推进")
	}
}

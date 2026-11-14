package engine

import (
	"context"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func TestEachServiceUsesRandomTimeWithinBounds(t *testing.T) {
	config := neutralMealConfig()
	config.Tick = 50 * time.Millisecond
	students := make([]agents.Student, 20)
	for i := range students {
		students[i] = mealStudent()
		students[i].ID = i + 1
		// 错开到达，使事件间隔只包含服务耗时。
		students[i].ArrivalTime = time.Duration(i) * 30 * time.Second
	}
	report, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.Completed != len(students) {
		t.Fatal("所有学生应完成取餐与付款")
	}
	for _, service := range []struct {
		name             string
		start, finish    agents.Signal
		minimum, maximum time.Duration
	}{
		{"自助取餐", agents.SelfServiceStart, agents.Entered, environment.MinSelfServiceTime, environment.MaxSelfServiceTime},
		{"主菜打饭", agents.OrderPlaced, agents.MainServed, environment.MinMainServiceTime, environment.MaxMainServiceTime},
		{"付款", agents.PaymentStarted, agents.Paid, environment.MinPaymentTime, environment.MaxPaymentTime},
	} {
		t.Run(service.name, func(t *testing.T) {
			starts := make(map[int]time.Duration)
			durations := make(map[time.Duration]bool)
			completed := 0
			// 服务完成事件向上对齐到逻辑时钟边界。
			upperBound := (service.maximum + config.Tick - 1) / config.Tick * config.Tick
			for _, event := range report.Events {
				if event.Signal == service.start {
					starts[event.StudentID] = event.Time
				}
				if event.Signal != service.finish || (event.Signal == agents.Entered && event.To != agents.MainService) {
					continue
				}
				start, ok := starts[event.StudentID]
				if !ok {
					t.Fatal("服务完成前必须有开始事件")
				}
				duration := event.Time - start
				if duration < service.minimum || duration > upperBound {
					t.Fatalf("服务耗时 %s 超出范围 [%s, %s]", duration, service.minimum, upperBound)
				}
				durations[duration] = true
				completed++
			}
			if completed != len(students) || len(durations) < 2 {
				t.Fatal("每名学生都应完成服务，多次服务应出现不同耗时")
			}
		})
	}
}

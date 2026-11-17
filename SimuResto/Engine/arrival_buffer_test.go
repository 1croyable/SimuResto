package engine

import (
	"context"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func TestArrivalBufferReleasesStudentsInOrder(t *testing.T) {
	for _, tick := range []time.Duration{250 * time.Millisecond, 700 * time.Millisecond, 2 * time.Second} {
		t.Run(tick.String(), func(t *testing.T) {
			students := make([]agents.Student, 4)
			for i := range students {
				students[i] = agents.Student{ID: i + 1, Preference: agents.Vegetarian, Satisfaction: 80, Patience: time.Hour}
			}
			// 输入顺序不等于到达顺序；同一计划时间内保持原顺序。
			students[0].ArrivalTime = 10 * time.Second
			config := environment.DefaultConfig()
			config.Tick = tick
			game, err := NewGame(config, students)
			if err != nil {
				t.Fatal(err)
			}
			defer game.Close()
			for game.engine.nextArrivalIndex < len(students) {
				if _, err := game.Step(context.Background()); err != nil {
					t.Fatal(err)
				}
				for _, student := range game.engine.students {
					if student.result.Status != agents.Scheduled {
						continue
					}
					if student.agent.in != nil || student.result.Satisfaction != student.result.Student.Satisfaction || student.result.TotalWait != 0 || len(student.result.Path) != 0 {
						t.Fatal("缓冲中的学生不能启动智能体、扣满意度或累计排队时间")
					}
				}
			}
			var arrivals []Event
			for _, event := range game.Report().Events {
				if event.Signal == agents.Arrived {
					arrivals = append(arrivals, event)
				}
			}
			if len(arrivals) != len(students) {
				t.Fatal("缓冲放行不能遗漏或重复学生")
			}
			for i, id := range []int{2, 3, 4, 1} {
				event := arrivals[i]
				if event.StudentID != id || event.Time < students[id-1].ArrivalTime {
					t.Fatal("必须按计划顺序放行，且不能早于计划时间")
				}
				if i > 0 && event.Time-arrivals[i-1].Time < environment.MinArrivalInterval {
					t.Fatal("实际抵达必须至少间隔一秒")
				}
				if event.Time%tick != 0 || game.engine.ID2Student[id].result.ArrivalTime != event.Time {
					t.Fatal("实际抵达时间必须对齐游戏刻度并写入结果")
				}
			}
			if tick == 250*time.Millisecond && (arrivals[0].Time != 0 || arrivals[1].Time != time.Second || arrivals[2].Time != 2*time.Second || arrivals[3].Time != 10*time.Second) {
				t.Fatal("默认刻度下前三人应在 0、1、2 秒到达，空闲后第四人在第 10 秒到达")
			}
		})
	}
}

func TestArrivalBufferDoesNotCatchUpInOneTick(t *testing.T) {
	students := []agents.Student{
		{ID: 1, Preference: agents.Vegetarian, Satisfaction: 80, Patience: time.Hour},
		{ID: 2, Preference: agents.Vegetarian, Satisfaction: 80, Patience: time.Hour},
		{ID: 3, Preference: agents.Vegetarian, Satisfaction: 80, Patience: time.Hour},
	}
	game, err := NewGame(environment.DefaultConfig(), students)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	game.engine.admit()
	game.engine.admit()
	if game.engine.nextArrivalIndex != 1 {
		t.Fatal("同一时刻重复调用也只能放行一人")
	}
	game.engine.now = 10 * time.Second
	game.engine.admit()
	game.engine.admit()
	if game.engine.nextArrivalIndex != 2 {
		t.Fatal("时间跳跃后也不能在同一刻度集中补发")
	}
}

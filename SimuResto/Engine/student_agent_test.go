package engine

import (
	"context"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func TestStudentAgentsStartOnlyAfterAdmission(t *testing.T) {
	// 开门缓冲人数超过队列容量，最后一名学生尚未到计划时间。
	count := environment.OutdoorCapacity + environment.IndoorCapacity + environment.SelfServiceCapacity + 2
	students := make([]agents.Student, count)
	for i := range students {
		students[i] = agents.Student{
			ID: i + 1, Preference: agents.Vegetarian,
			Satisfaction: 80, Patience: time.Hour,
		}
	}
	students[count-1].ArrivalTime = time.Hour

	for _, admit := range []bool{false, true} {
		name := "入场前关闭"
		if admit {
			name = "入场后关闭"
		}
		t.Run(name, func(t *testing.T) {
			game, err := NewGame(environment.DefaultConfig(), students)
			if err != nil {
				t.Fatal(err)
			}
			defer game.Close()
			for _, student := range game.engine.students {
				if student.agent.in != nil || student.agent.out != nil {
					t.Fatal("提前创建资料不应启动学生智能体")
				}
			}
			if !admit {
				return
			}

			// 只放行、不推进服务，直到队列填满；缓冲区学生按秒依次抵达。
			for i := 0; i < count-1; i++ {
				game.engine.now = time.Duration(i) * environment.MinArrivalInterval
				game.engine.admit()
			}
			active, abandoned, scheduled := 0, 0, 0
			for _, student := range game.engine.students {
				switch student.result.Status {
				case agents.Active:
					active++
					if student.agent.in == nil || student.agent.out == nil || len(student.result.Path) == 0 {
						t.Fatal("成功入队的学生必须启动智能体")
					}
				case agents.Abandoned:
					abandoned++
				case agents.Scheduled:
					scheduled++
				}
				if student.result.Status != agents.Active && (student.agent.in != nil || student.agent.out != nil) {
					t.Fatal("未到达或直接放弃的学生不应启动智能体")
				}
			}
			if active != count-2 || abandoned != 1 || scheduled != 1 {
				t.Fatalf("入队 %d、直接放弃 %d、未到达 %d，数量不符合预期", active, abandoned, scheduled)
			}
			firstAgent := game.engine.students[0].agent
			if _, err := game.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
			if game.engine.students[0].agent != firstAgent {
				t.Fatal("后续推进不应重复创建学生智能体")
			}
		})
	}
}

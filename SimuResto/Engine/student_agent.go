package engine

import (
	"math/rand"
	"time"

	agents "simuresto/SimuResto/Agents"
)

type observation struct {
	area         agents.Area
	waiting      time.Duration
	satisfaction float32
	dishes       []agents.Dish // 本次领取时有库存且符合饮食限制的菜品。
}

type decision struct {
	leave  bool
	dishID string // 本次抽中的一份菜品。
}

type studentAgent struct {
	in  chan observation
	out chan decision
}

// 每名学生拥有独立 goroutine 和随机源，按实时可用菜品抽取一份。
func runStudent(student agents.Student, seed int64, agent studentAgent) {
	studentSeed := int64(uint64(seed) ^ (uint64(student.ID) * 0x9e3779b97f4a7c15))
	rng := rand.New(rand.NewSource(studentSeed))
	for state := range agent.in {
		response := decision{}
		// 领取托盘后，学生继续完成取餐和付款，不再因耐心耗尽离队。
		response.leave = state.area.IsWaitingArea() &&
			(state.waiting >= student.Patience || state.satisfaction <= 0)
		response.dishID = weightedChoice(rng, state.dishes, student.Preference, student.Temperature)
		agent.out <- response
	}
}

// decisionScope 明确区分全体学生决策与仅排队学生的离队决策。
type decisionScope uint8

const (
	allStudents decisionScope = iota
	waitingStudents
)

func (e *engine) askStudents(scope decisionScope) {
	var asked []*studentState
	for _, student := range e.students {
		if student.result.Status != agents.Active {
			continue
		}
		area := student.result.Area
		if scope == waitingStudents && !area.IsWaitingArea() {
			continue
		}
		student.agent.in <- observation{
			area: area, waiting: student.result.TotalWait, satisfaction: student.result.Satisfaction,
		}
		asked = append(asked, student)
	}
	for _, student := range asked {
		response := <-student.agent.out
		if response.leave {
			area := student.result.Area
			e.queues[area].Remove(student.result.Student.ID)
			student.result.Area = agents.NoArea
			student.result.Status = agents.Abandoned
			student.result.ExitTime = e.now
			e.stats.finishedStudentCount++
			e.emit(agents.LeftQueue, student.result.Student.ID, 0, area, agents.NoArea)
		}
	}
}

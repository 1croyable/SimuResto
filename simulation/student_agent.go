package simulation

import "time"

type observation struct {
	area         Area
	waiting      time.Duration
	satisfaction float32
}

type decision struct {
	leave          bool
	starterChoices []string
	mainChoices    []string // 按偏好排序的候选项，最终只选择一份有库存的主菜。
	dessertChoices []string
}

type studentAgent struct {
	in  chan observation
	out chan decision
}

// 每名学生拥有独立 goroutine，返回按偏好排列的候选菜品；
// 协调器从中选择第一个有库存的菜品，并原子地更新库存。
func runStudent(student Student, menu []Dish, seed int64, agent studentAgent) {
	plan := chooseMeal(student, menu, seed)
	for state := range agent.in {
		response := plan
		// 领取托盘后，学生继续完成取餐和付款，不再因耐心耗尽离队。
		response.leave = state.area.IsWaitingArea() &&
			(state.waiting >= student.Patience || state.satisfaction <= 0)
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
		if student.result.Status != Active {
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
		student.plan = <-student.agent.out
		if student.plan.leave {
			area := student.result.Area
			e.queues[area].Remove(student.result.Student.ID)
			student.result.Area = NoArea
			student.result.Status = Abandoned
			student.result.ExitTime = e.now
			e.finished++
			e.emit(LeftQueue, student.result.Student.ID, 0, area, NoArea)
		}
	}
}

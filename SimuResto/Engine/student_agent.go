package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type observation struct {
	area           agents.Area
	waiting        time.Duration
	satisfaction   float32
	outdoorCrowded bool          // 室外人数是否超过允许放弃排队的阈值。
	dishes         []agents.Dish // 本次领取时有库存且符合饮食限制的菜品。
}

type decision struct {
	leave  bool
	dishID string // 本次抽中的一份菜品。
}

type studentAgent struct {
	in  chan observation
	out chan decision
}

// 学生成功加入队列后才创建通信通道并启动 goroutine。
func (e *engine) startStudentAgent(student *studentState) {
	student.agent = studentAgent{in: make(chan observation), out: make(chan decision)}
	profile, agent := student.result.Student, student.agent
	e.agentWG.Add(1)
	go func() {
		defer e.agentWG.Done()
		runStudent(profile, agent)
	}()
}

// 每名学生在自己的 goroutine 中，按实时可用菜品随机抽取一份。
func runStudent(student agents.Student, agent studentAgent) {
	for state := range agent.in {
		response := decision{}
		// 只有室外较拥挤时才考虑离队；进入室内后始终完成取餐与付款。
		response.leave = state.area == agents.Outside && state.outdoorCrowded &&
			(state.waiting >= student.Patience || state.satisfaction <= 0)
		response.dishID = weightedChoice(state.dishes, student.Preference, student.Temperature)
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
	outdoorCrowded := e.outdoorQueueCrowded()
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
			outdoorCrowded: outdoorCrowded,
		}
		asked = append(asked, student)
	}
	for _, student := range asked {
		response := <-student.agent.out
		// 多名学生可能同时提出离队；逐个执行时重新检查人数，降到阈值即停止。
		if response.leave && student.result.Area == agents.Outside && e.outdoorQueueCrowded() {
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

func (e *engine) outdoorQueueCrowded() bool {
	queue := e.queues[agents.Outside]
	return queue.Len()*100 > queue.Capacity()*environment.OutdoorAbandonmentThresholdPercent
}

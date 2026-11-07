package engine

import (
	"sort"
	"sync"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type studentState struct {
	result       StudentResult
	agent        studentAgent
	started      bool          // 当前区域的服务已经开始。
	ready        bool          // 当前服务已完成，等待下一区域腾出位置。
	elapsed      time.Duration // 在自助区已经花费的取餐时间。
	missingSides int           // 最终缺少的配菜份数，与整餐评分统一结算。
}

type job struct {
	studentID int
	dishID    string
	fresh     bool
}

type staffState struct {
	result StaffResult
	agent  staffAgent
	job    *job
}

type simulationStats struct {
	finishedStudentCount int                 // 已结束的学生总数：完成、离队、拒绝入场或取消均计入。
	peakOccupancy        map[agents.Area]int // 每个区域曾同时容纳的最大学生人数。
	events               []Event             // 按发生顺序保存的事件记录。
}

// 协调器独占队列、库存和结果状态的写入权限。
// 智能体通过 channel 通信。
type engine struct {
	config environment.Config // 本轮模拟的规则、容量、服务时长和菜单。
	now    time.Duration      // 从模拟开始累计经过的模拟时间，不是现实中的钟表时间。

	students         []*studentState       // 全体学生的运行状态，保留输入顺序，包含未到达及已结束的学生。
	ID2Student       map[int]*studentState // 用学生 ID 快速找到运行状态，与 students 指向同一批对象。
	arrivalStudent   []*studentState       // 按计划到达时间排序的学生列表，与 students 共享状态对象。
	nextArrivalIndex int                   // arrivalStudent 中下一条尚未处理的到达记录的下标。

	staff  []*staffState                      // 打饭人员和收银员的运行状态，包含当前任务和通信 channel。
	queues map[agents.Area]*environment.Queue // 每个区域的 FIFO 队列，保存学生 ID；包括正在接受服务的学生。
	stock  *inventory                         // 本轮菜品库存，管理领取、主菜预留、兜底补货和归还。

	stats   simulationStats // 本轮累计的统计数据和事件记录。
	agentWG sync.WaitGroup  // 等待所有学生和工作人员 goroutine 退出。
}

func newEngine(config environment.Config, students []agents.Student) *engine {
	config.Menu = append([]agents.Dish(nil), config.Menu...)
	for i := range config.Menu {
		config.Menu[i] = config.Menu[i].Prepared()
	}
	e := &engine{
		config:     config,
		ID2Student: make(map[int]*studentState),
		queues:     make(map[agents.Area]*environment.Queue),
		stock:      newInventory(config.Menu),
		stats:      simulationStats{peakOccupancy: make(map[agents.Area]int)},
	}
	capacities := []int{
		config.Capacities.Outside, config.Capacities.Inside, config.Capacities.SelfService,
		config.Capacities.MainService, config.Capacities.Payment,
	}
	for i, area := range agents.ServiceRoute() {
		e.queues[area], _ = environment.NewQueue(capacities[i]) // 容量已在启动前校验。
		e.stats.peakOccupancy[area] = 0
	}
	for _, student := range students {
		state := &studentState{
			result: StudentResult{
				Student: student, Status: agents.Scheduled, Satisfaction: student.Satisfaction,
				Waiting: make(map[agents.Area]time.Duration),
			},
			agent: studentAgent{in: make(chan observation), out: make(chan decision)},
		}
		e.students = append(e.students, state)
		e.ID2Student[student.ID] = state
		e.arrivalStudent = append(e.arrivalStudent, state)
		e.agentWG.Add(1)
		go func() {
			defer e.agentWG.Done()
			runStudent(student, config.Seed, state.agent)
		}()
	}

	sort.SliceStable(e.arrivalStudent, func(i, j int) bool {
		return e.arrivalStudent[i].result.Student.ArrivalTime < e.arrivalStudent[j].result.Student.ArrivalTime
	})

	for i := 0; i < config.Servers+config.Cashiers; i++ {
		staff := agents.Staff{ID: i + 1, Role: agents.Server, ServiceTime: config.MainServiceTime}
		if i >= config.Servers {
			staff.Role, staff.ServiceTime = agents.Cashier, config.PaymentTime
		}
		state := &staffState{
			result: StaffResult{Staff: staff},
			agent:  staffAgent{in: make(chan workTick), out: make(chan workResult)},
		}
		e.staff = append(e.staff, state)
		e.agentWG.Add(1)
		go func() {
			defer e.agentWG.Done()
			runStaff(staff, state.agent)
		}()
	}
	return e
}

func (e *engine) closeAgents() {
	for _, student := range e.students {
		close(student.agent.in)
	}
	for _, staff := range e.staff {
		close(staff.agent.in)
	}
	e.agentWG.Wait()
}

func (e *engine) stop() {
	for _, worker := range e.staff {
		if worker.job != nil && worker.job.dishID != "" {
			e.stock.releaseMain(worker.job.dishID)
		}
		worker.job = nil
	}
	for _, student := range e.students {
		if student.result.Status != agents.Active && student.result.Status != agents.Scheduled {
			continue
		}
		area := student.result.Area
		if area != agents.NoArea {
			e.queues[area].Remove(student.result.Student.ID)
		}
		student.result.Area = agents.NoArea
		student.result.Status = agents.Cancelled
		student.result.ExitTime = e.now
		e.stats.finishedStudentCount++
		e.emit(agents.Stopped, student.result.Student.ID, 0, area, agents.NoArea)
	}
}

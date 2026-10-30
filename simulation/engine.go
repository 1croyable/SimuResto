package simulation

import (
	"sort"
	"sync"
	"time"
)

type studentState struct {
	result  StudentResult
	agent   studentAgent
	plan    decision
	started bool          // 当前区域的服务已经开始。
	ready   bool          // 当前服务已完成，等待下一区域腾出位置。
	elapsed time.Duration // 在自助区已经花费的取餐时间。
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

// 协调器独占队列、库存和结果状态的写入权限。
// 智能体通过 channel 通信。
type engine struct {
	config   Config
	now      time.Duration
	students []*studentState
	byID     map[int]*studentState
	arrivals []*studentState
	next     int
	finished int
	staff    []*staffState
	queues   map[Area]*Queue
	stock    *inventory
	events   []Event
	peaks    map[Area]int
	agents   sync.WaitGroup
}

func newEngine(config Config, students []Student) *engine {
	// 复制菜单，避免持有调用方切片的底层数组。
	config.Menu = append([]Dish(nil), config.Menu...)
	e := &engine{
		config: config, byID: make(map[int]*studentState), queues: make(map[Area]*Queue),
		stock: newInventory(config.Menu), peaks: make(map[Area]int),
	}
	capacities := []int{
		config.Capacities.Outside, config.Capacities.Inside, config.Capacities.SelfService,
		config.Capacities.MainService, config.Capacities.Payment,
	}
	for i, area := range ServiceRoute() {
		e.queues[area], _ = NewQueue(capacities[i]) // 容量已在启动前校验。
		e.peaks[area] = 0
	}
	for _, student := range students {
		state := &studentState{
			result: StudentResult{
				Student: student, Status: Scheduled, Satisfaction: student.Satisfaction,
				Waiting: make(map[Area]time.Duration),
			},
			agent: studentAgent{in: make(chan observation), out: make(chan decision)},
		}
		e.students = append(e.students, state)
		e.byID[student.ID] = state
		e.arrivals = append(e.arrivals, state)
		e.agents.Add(1)
		go func() {
			defer e.agents.Done()
			runStudent(student, config.Menu, config.Seed, state.agent)
		}()
	}
	sort.SliceStable(e.arrivals, func(i, j int) bool {
		return e.arrivals[i].result.Student.ArrivalTime < e.arrivals[j].result.Student.ArrivalTime
	})
	for i := 0; i < config.Servers+config.Cashiers; i++ {
		staff := Staff{ID: i + 1, Role: Server, ServiceTime: config.MainServiceTime}
		if i >= config.Servers {
			staff.Role, staff.ServiceTime = Cashier, config.PaymentTime
		}
		state := &staffState{
			result: StaffResult{Staff: staff},
			agent:  staffAgent{in: make(chan workTick), out: make(chan workResult)},
		}
		e.staff = append(e.staff, state)
		e.agents.Add(1)
		go func() {
			defer e.agents.Done()
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
	e.agents.Wait()
}

func (e *engine) stop() {
	for _, worker := range e.staff {
		if worker.job != nil && worker.job.dishID != "" {
			e.stock.releaseMain(worker.job.dishID)
		}
		worker.job = nil
	}
	for _, student := range e.students {
		if student.result.Status != Active && student.result.Status != Scheduled {
			continue
		}
		area := student.result.Area
		if area != NoArea {
			e.queues[area].Remove(student.result.Student.ID)
		}
		student.result.Area = NoArea
		student.result.Status = Cancelled
		student.result.ExitTime = e.now
		e.finished++
		e.emit(Stopped, student.result.Student.ID, 0, area, NoArea)
	}
}

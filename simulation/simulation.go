package simulation

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrTimeLimit = errors.New("simulation reached its maximum duration")

type studentState struct {
	result  StudentResult
	agent   studentAgent
	plan    decision
	started bool
	ready   bool
	elapsed time.Duration
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

// The coordinator is the sole owner of queues, inventory, and reported state.
// Agents communicate over channels; no goroutine directly shares mutable state.
type engine struct {
	config    Config
	now       time.Duration
	students  []*studentState
	byID      map[int]*studentState
	arrivals  []*studentState
	next      int
	finished  int
	staff     []*staffState
	queues    map[Area]*Queue
	stock     []Stock
	stockByID map[string]int
	events    []Event
	peaks     map[Area]int
	agents    sync.WaitGroup
}

// Run starts one goroutine per student and staff member. Tick is logical time;
// TickInterval optionally slows playback. Identical inputs produce the same
// report regardless of goroutine scheduling. A cancellation or time limit
// returns a partial report, releases reservations, and joins every agent.
func Run(ctx context.Context, config Config, students []Student) (Report, error) {
	if err := config.validate(students); err != nil {
		return Report{}, err
	}
	e := newEngine(config, students)
	defer e.closeAgents()
	for {
		if err := ctx.Err(); err != nil {
			e.stop()
			return e.report(), err
		}
		e.moveForward()
		e.admit()
		if e.finished == len(e.students) {
			return e.report(), nil
		}
		if e.now >= config.MaxDuration {
			e.stop()
			return e.report(), ErrTimeLimit
		}

		e.askStudents(false)
		e.startSelfService()
		e.assignStaff()
		step := min(config.Tick, config.MaxDuration-e.now)
		e.countWaiting(step)
		e.now += step
		e.advanceService(step)
		// Give waiting students a chance to leave before newly freed places
		// are offered at the next clock boundary.
		e.askStudents(true)
		e.moveForward()
		if e.finished == len(e.students) {
			return e.report(), nil
		}

		if config.TickInterval > 0 && e.now < config.MaxDuration {
			timer := time.NewTimer(config.TickInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				e.stop()
				return e.report(), ctx.Err()
			case <-timer.C:
			}
		}
	}
}

func newEngine(config Config, students []Student) *engine {
	// Menu backing arrays supplied by the caller are not retained.
	config.Menu = append([]Dish(nil), config.Menu...)
	e := &engine{
		config: config, byID: make(map[int]*studentState), queues: make(map[Area]*Queue),
		stockByID: make(map[string]int), peaks: make(map[Area]int),
	}
	capacities := []int{
		config.Capacities.Outside, config.Capacities.Inside, config.Capacities.SelfService,
		config.Capacities.MainService, config.Capacities.Payment,
	}
	for i, area := range route {
		e.queues[area], _ = NewQueue(capacities[i]) // Already validated.
		e.peaks[area] = 0
	}
	for _, dish := range config.Menu {
		e.stockByID[dish.ID] = len(e.stock)
		e.stock = append(e.stock, Stock{Dish: dish, Remaining: dish.Portions})
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
			runStudent(student, config.Menu, state.agent)
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

func (e *engine) emit(signal Signal, id, staffID int, from, to Area, dishes ...string) {
	e.events = append(e.events, Event{
		Time: e.now, Signal: signal, StudentID: id, StaffID: staffID,
		From: from, To: to, DishIDs: dishes,
	})
}

func (e *engine) enter(student *studentState, area Area) {
	id, from := student.result.Student.ID, student.result.Area
	if !e.queues[area].Enqueue(id) {
		panic("simulation: transfer into a full queue")
	}
	student.result.Area = area
	student.result.Path = append(student.result.Path, area)
	student.started, student.ready, student.elapsed = false, false, 0
	e.peaks[area] = max(e.peaks[area], e.queues[area].Len())
	e.emit(Entered, id, 0, from, area)
}

func (e *engine) admit() {
	for e.next < len(e.arrivals) && e.arrivals[e.next].result.Student.ArrivalTime <= e.now {
		student := e.arrivals[e.next]
		e.next++
		student.result.ArrivalTime = e.now
		e.emit(Arrived, student.result.Student.ID, 0, "", "")
		area := Outside
		if e.queues[Outside].Len() == 0 && e.queues[Inside].Len() < e.queues[Inside].Capacity() {
			area = Inside
		}
		if e.queues[area].Len() == e.queues[area].Capacity() {
			student.result.Status = Rejected
			student.result.ExitTime = e.now
			e.finished++
			e.emit(QueueFull, student.result.Student.ID, 0, "", "")
			continue
		}
		student.result.Status = Active
		e.enter(student, area)
		// Immediate transfers let simultaneous arrivals use all free places,
		// while the existing outdoor queue always has priority.
		e.moveForward()
	}
}

func (e *engine) moveForward() {
	for {
		changed := false
		for i := len(route) - 1; i >= 0; i-- {
			area, queue := route[i], e.queues[route[i]]
			for {
				id, ok := queue.Front()
				if !ok {
					break
				}
				student := e.byID[id]
				if area != Outside && area != Inside && !student.ready {
					break
				}
				if area == Payment {
					queue.Dequeue()
					student.result.Area = ""
					student.result.Status = Completed
					student.result.ExitTime = e.now
					e.finished++
					e.emit(Finished, id, 0, Payment, "")
				} else {
					next := route[i+1]
					if e.queues[next].Len() == e.queues[next].Capacity() {
						break
					}
					queue.Dequeue()
					e.enter(student, next)
				}
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

func (e *engine) askStudents(waitingOnly bool) {
	var asked []*studentState
	for _, student := range e.students {
		if student.result.Status != Active {
			continue
		}
		area := student.result.Area
		if waitingOnly && area != Outside && area != Inside {
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
			student.result.Area = ""
			student.result.Status = Abandoned
			student.result.ExitTime = e.now
			e.finished++
			e.emit(LeftQueue, student.result.Student.ID, 0, area, "")
		}
	}
}

func (e *engine) adjust(student *studentState, amount float64) {
	student.result.Satisfaction = min(100, max(0, student.result.Satisfaction+amount))
}

func (e *engine) takeSides(choices []string, count int) []Dish {
	var dishes []Dish
	for range count {
		found := false
		for _, id := range choices {
			stock := &e.stock[e.stockByID[id]]
			if stock.Remaining > 0 {
				stock.Remaining--
				stock.Taken++
				dishes = append(dishes, stock.Dish)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return dishes
}

func (e *engine) startSelfService() {
	for _, id := range e.queues[SelfService].IDs() {
		student := e.byID[id]
		if student.started {
			continue
		}
		student.started = true
		e.emit(SelfServiceStart, id, 0, SelfService, SelfService)
		wantStarters := min(student.result.Student.StartersWanted, e.config.MaxStartersPerStudent, e.config.MaxSelfServiceItems)
		student.result.Meal.Starters = e.takeSides(student.plan.starters, wantStarters)
		wantDesserts := min(student.result.Student.DessertsWanted, e.config.MaxDessertsPerStudent,
			e.config.MaxSelfServiceItems-len(student.result.Meal.Starters))
		student.result.Meal.Desserts = e.takeSides(student.plan.desserts, wantDesserts)
		missing := wantStarters + wantDesserts - len(student.result.Meal.Starters) - len(student.result.Meal.Desserts)
		var dishes []string
		bonus := 0.0
		for _, course := range [][]Dish{student.result.Meal.Starters, student.result.Meal.Desserts} {
			if len(course) > 0 {
				bonus += e.config.CourseVarietyBonus
			}
			for _, dish := range course {
				dishes = append(dishes, dish.ID)
			}
		}
		e.adjust(student, bonus-float64(missing)*e.config.MissingSidePenalty)
		e.emit(FoodCollected, id, 0, SelfService, SelfService, dishes...)
		if missing > 0 {
			e.emit(FoodUnavailable, id, 0, SelfService, SelfService)
		}
	}
}

func (e *engine) reserveMain(choices []string) string {
	for _, id := range choices {
		stock := &e.stock[e.stockByID[id]]
		if stock.Remaining > 0 {
			stock.Remaining--
			stock.Reserved++
			return id
		}
	}
	return ""
}

func (e *engine) assignStaff() {
	for _, worker := range e.staff {
		if worker.job != nil {
			continue
		}
		area := MainService
		if worker.result.Staff.Role == Cashier {
			area = Payment
		}
		for _, id := range e.queues[area].IDs() {
			student := e.byID[id]
			if student.started {
				continue
			}
			student.started = true
			task := &job{studentID: id, fresh: true}
			signal := PaymentStarted
			if area == MainService {
				task.dishID = e.reserveMain(student.plan.mains)
				if task.dishID == "" {
					student.ready = true
					e.adjust(student, -e.config.MissingMainPenalty)
					e.emit(FoodUnavailable, id, 0, area, area)
					continue
				}
				signal = OrderPlaced
			}
			worker.job = task
			var dishes []string
			if task.dishID != "" {
				dishes = []string{task.dishID}
			}
			e.emit(signal, id, worker.result.Staff.ID, area, area, dishes...)
			break
		}
	}
}

func (e *engine) countWaiting(step time.Duration) {
	for _, student := range e.students {
		if student.result.Status != Active {
			continue
		}
		area := student.result.Area
		if area != Outside && area != Inside && student.started && !student.ready {
			continue // Time spent collecting food or being served is not queueing.
		}
		student.result.Waiting[area] += step
		student.result.TotalWait += step
		loss := e.config.IndoorLossPerSecond
		if area == Outside {
			loss = e.config.OutdoorLossPerSecond
		}
		e.adjust(student, -loss*step.Seconds())
	}
}

func (e *engine) advanceService(step time.Duration) {
	for _, id := range e.queues[SelfService].IDs() {
		student := e.byID[id]
		if student.started && !student.ready {
			student.elapsed += step
			if student.elapsed >= e.config.SelfServiceTime {
				student.ready = true
			}
		}
	}
	// Send all workers their clock update before collecting replies. Applying
	// replies in staff order makes the event log independent of the scheduler.
	for _, worker := range e.staff {
		if worker.job == nil {
			continue
		}
		update := workTick{elapsed: step}
		if worker.job.fresh {
			update.start = &assignment{studentID: worker.job.studentID, duration: worker.result.Staff.ServiceTime}
			worker.job.fresh = false
		}
		worker.agent.in <- update
	}
	for _, worker := range e.staff {
		if worker.job == nil {
			continue
		}
		result := <-worker.agent.out
		worker.result.BusyTime += result.busy
		if result.signal == Progress {
			continue
		}
		student := e.byID[result.studentID]
		student.ready = true
		worker.result.Served++
		var dishes []string
		if result.signal == MainServed {
			stock := &e.stock[e.stockByID[worker.job.dishID]]
			stock.Reserved--
			stock.Taken++
			dish := stock.Dish
			student.result.Meal.Main = &dish
			dishes = []string{dish.ID}
			change := e.config.PreferredMainBonus
			if student.result.Student.Preference == MeatPreferred && dish.Vegetarian {
				change = -e.config.AlternativeMainPenalty
			}
			e.adjust(student, change+e.config.CourseVarietyBonus)
		}
		e.emit(result.signal, result.studentID, worker.result.Staff.ID, student.result.Area, student.result.Area, dishes...)
		worker.job = nil
	}
}

func (e *engine) stop() {
	for _, worker := range e.staff {
		if worker.job != nil && worker.job.dishID != "" {
			stock := &e.stock[e.stockByID[worker.job.dishID]]
			stock.Reserved--
			stock.Remaining++
		}
		worker.job = nil
	}
	for _, student := range e.students {
		if student.result.Status != Active && student.result.Status != Scheduled {
			continue
		}
		area := student.result.Area
		if area != "" {
			e.queues[area].Remove(student.result.Student.ID)
		}
		student.result.Area = ""
		student.result.Status = Cancelled
		student.result.ExitTime = e.now
		e.finished++
		e.emit(Stopped, student.result.Student.ID, 0, area, "")
	}
}

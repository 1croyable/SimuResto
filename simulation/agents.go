package simulation

import "time"

type observation struct {
	area         Area
	waiting      time.Duration
	satisfaction float64
}

type decision struct {
	leave    bool
	starters []string
	mains    []string
	desserts []string
}

type studentAgent struct {
	in  chan observation
	out chan decision
}

// Each student has its own goroutine. Plans contain alternatives in preference
// order; the coordinator atomically chooses the first available dish.
func runStudent(student Student, menu []Dish, agent studentAgent) {
	plan := decision{}
	for _, dish := range menu {
		if student.Preference == Vegetarian && !dish.Vegetarian {
			continue
		}
		switch dish.Course {
		case Starter:
			plan.starters = append(plan.starters, dish.ID)
		case Main:
			if student.Preference == Vegetarian || !dish.Vegetarian {
				plan.mains = append(plan.mains, dish.ID)
			}
		case Dessert:
			plan.desserts = append(plan.desserts, dish.ID)
		}
	}
	if student.Preference == MeatPreferred {
		for _, dish := range menu {
			if dish.Course == Main && dish.Vegetarian {
				plan.mains = append(plan.mains, dish.ID)
			}
		}
	}
	for state := range agent.in {
		response := plan
		// Once a tray is collected, the student finishes the service route.
		response.leave = (state.area == Outside || state.area == Inside) &&
			(state.waiting >= student.Patience || state.satisfaction <= 0)
		agent.out <- response
	}
}

type assignment struct {
	studentID int
	duration  time.Duration
}

type workTick struct {
	start   *assignment
	elapsed time.Duration
}

type workResult struct {
	signal    Signal
	studentID int
	busy      time.Duration
}

type staffAgent struct {
	in  chan workTick
	out chan workResult
}

// A worker keeps its own current task and remaining service time.
func runStaff(staff Staff, agent staffAgent) {
	var current assignment
	var remaining time.Duration
	for tick := range agent.in {
		if tick.start != nil {
			current = *tick.start
			remaining = current.duration
		}
		busy := min(tick.elapsed, remaining)
		remaining -= busy
		signal := Progress
		if remaining == 0 {
			signal = MainServed
			if staff.Role == Cashier {
				signal = Paid
			}
		}
		agent.out <- workResult{signal: signal, studentID: current.studentID, busy: busy}
	}
}

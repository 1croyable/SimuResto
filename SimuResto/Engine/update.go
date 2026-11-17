package engine

import (
	"math"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

// stores temporary information needed during one simulation step
type queueTick struct {
	route []agents.Point

	starter1End int
	starter2End int
	mainEnd     int

	starter1Position float64
	starter2Position float64
	mainPosition     float64
	paymentPosition  float64

	foodCounterFreeAt    time.Duration
	paymentCounterFreeAt time.Duration

	dt time.Duration
}

func (engine *engine) advanceQueue(world *environment.Environment, dt time.Duration) {
	if world.Queue == nil || dt <= 0 {
		return
	}

	route := append([]agents.Point(nil), world.Route...)
	route = append(route, world.Starter2Route...)
	route = append(route, world.MainRoute...)
	route = append(route, world.CheckoutRoute...)

	q := queueTick{
		route:       route,
		starter1End: len(world.Route),
		dt:          dt,
	}

	q.starter2End = q.starter1End + len(world.Starter2Route)
	q.mainEnd = q.starter2End + len(world.MainRoute)

	q.starter1Position = routeLength(route[:q.starter1End])
	q.starter2Position = routeLength(route[:q.starter2End])
	q.mainPosition = routeLength(route[:q.mainEnd])
	q.paymentPosition = routeLength(route)

	personAheadPosition := q.paymentPosition + world.Settings.QueueSpacing
	peopleAhead := 0

	// calculate one by one from the front
	for _, id := range world.Queue.IDs() {
		student := world.StudentsByID[id]
		remainingTime := min(dt, world.TimePassed-student.Student.ArrivalTime)
		leftQueue := false

		for remainingTime > 0 {
			switch student.Status {

			case agents.Walking, agents.WaitingForFood, agents.WaitingForPayment:
				leftQueue = engine.advanceTowardCounter(world, student, id, &q, &remainingTime, personAheadPosition, peopleAhead)

			case agents.Serving:
				advanceFoodService(world, student, &q, &remainingTime)

			case agents.Paying:
				leftQueue = advancePayment(world, student, id, &q, &remainingTime)

			default:
				remainingTime = 0
			}
		}

		if !leftQueue {
			personAheadPosition = student.PositionAlongRoute
			peopleAhead++
		}
	}
}

// Handle walking, queue spacing, waiting, and starting a service.
func (engine *engine) advanceTowardCounter(
	world *environment.Environment,
	student *agents.StudentState,
	id int,
	tickInfo *queueTick,
	remainingTime *time.Duration,
	personAheadPosition float64,
	peopleAhead int,
) bool {

	beforeFood := student.Target == agents.TargetMain
	pickingStarter := student.Target == agents.TargetStarter1 ||
		student.Target == agents.TargetStarter2

	counter := &world.Cashier
	counterPosition := tickInfo.paymentPosition
	counterFreeAt := tickInfo.paymentCounterFreeAt

	switch student.Target {
	case agents.TargetMain:
		counter = &world.Cook
		counterPosition = tickInfo.mainPosition
		counterFreeAt = tickInfo.foodCounterFreeAt

	case agents.TargetStarter1:
		counterPosition = tickInfo.starter1Position
		counterFreeAt = 0

	case agents.TargetStarter2:
		counterPosition = tickInfo.starter2Position
		counterFreeAt = 0
	}

	// Move while maintaining spacing from the student ahead.
	stoppingPosition := min(
		counterPosition,
		personAheadPosition-world.Settings.QueueSpacing,
	)

	distance := max(0, stoppingPosition-student.PositionAlongRoute)

	timeToStop := time.Duration(math.Round(
		distance / world.Settings.WalkingSpeed * float64(time.Second),
	))

	walkingTime := min(*remainingTime, timeToStop)

	student.Walk(tickInfo.route, world.Settings.WalkingSpeed, walkingTime)
	student.PositionAlongRoute = min(
		stoppingPosition,
		student.PositionAlongRoute+world.Settings.WalkingSpeed*walkingTime.Seconds(),
	)

	*remainingTime -= walkingTime

	atCounter := student.PositionAlongRoute >= counterPosition-0.0001

	// Plan and reserve starters shortly before reaching counter 1.
	if student.Target == agents.TargetStarter1 &&
		student.PlannedStarterIDs == nil &&
		student.PositionAlongRoute >= tickInfo.starter1Position-20 {

		student.PlanStarters(world.Menu)

		for _, dishID := range student.PlannedStarterIDs {
			for i := range world.Menu {
				dish := &world.Menu[i]

				if dish.Course == agents.Starter && dish.ID == dishID {
					dish.Portions--
					break
				}
			}
		}
	}

	starterCount := 0

	if atCounter && pickingStarter {
		starterCount = countPlannedStarters(world, student)

		// Skip a starter counter if nothing was selected there.
		if starterCount == 0 {
			if student.Target == agents.TargetStarter1 {
				student.Target = agents.TargetStarter2
			} else {
				student.Target = agents.TargetMain
			}

			student.Status = agents.Walking
			student.QueueWait = 0
			return false
		}
	}

	var dishes []agents.DishOption
	canStartService := false

	switch {
	case pickingStarter:
		canStartService = atCounter

	case beforeFood:
		if atCounter && counter.StudentID == 0 {
			for _, dish := range world.Menu {
				if dish.Course == agents.Main && dish.Portions > 0 {
					dishes = append(dishes, agents.DishOption{
						ID: dish.ID,
					})
				}
			}

			canStartService = len(dishes) > 0
		}

	default:
		canStartService = atCounter && counter.StudentID == 0
	}

	waitingTime := *remainingTime

	if canStartService {
		timeAlreadyUsed := tickInfo.dt - *remainingTime

		if counterFreeAt > timeAlreadyUsed {
			waitingTime = min(*remainingTime, counterFreeAt-timeAlreadyUsed)
		} else {
			waitingTime = 0
		}
	}

	student.QueueWait += waitingTime
	*remainingTime -= waitingTime

	// Ask the student's goroutine whether to abandon the queue.
	decision := engine.requestStudentDecision(
		student,
		agents.StudentObservation{
			CanLeave:    student.Target != agents.TargetCashier,
			PeopleAhead: peopleAhead,
			Waiting:     student.QueueWait,
			Elapsed:     waitingTime,
		},
	)

	if decision.Leave {
		restoreReservedStarters(world, student)

		student.Status = agents.Abandoned
		world.Queue.Remove(id)
		*remainingTime = 0

		return true
	}

	if !canStartService || *remainingTime == 0 {
		switch {
		case student.Target == agents.TargetCashier:
			student.Status = agents.WaitingForPayment
		case beforeFood:
			student.Status = agents.WaitingForFood
		}

		*remainingTime = 0
		return false
	}

	// Begin the appropriate service.
	student.PositionAlongRoute = counterPosition

	switch {
	case pickingStarter:
		if student.Target == agents.TargetStarter1 {
			student.Position = tickInfo.route[tickInfo.starter1End-1]
			student.NextWaypoint = tickInfo.starter1End
		} else {
			student.Position = tickInfo.route[tickInfo.starter2End-1]
			student.NextWaypoint = tickInfo.starter2End
		}

		student.Status = agents.Serving
		student.RemainingStarterPickup =
			world.Settings.ServiceTime * time.Duration(starterCount)

	case beforeFood:
		// Ask the same goroutine to select a main course.
		choice := engine.requestStudentDecision(
			student,
			agents.StudentObservation{Dishes: dishes},
		)

		student.OrderedDishID = choice.DishID
		student.Position = world.MainRoute[len(world.MainRoute)-1]
		student.NextWaypoint = tickInfo.mainEnd
		student.Status = agents.Serving

	default:
		student.Position = tickInfo.route[len(tickInfo.route)-1]
		student.NextWaypoint = len(tickInfo.route)
		student.Status = agents.Paying
	}

	if !pickingStarter {
		counter.StudentID = id
		counter.ServiceTimeLeft = counter.Staff.ServiceTime
	}

	return false
}

// Handle starter collection and main-course service.
func advanceFoodService(world *environment.Environment, student *agents.StudentState, tickInfo *queueTick, remainingTime *time.Duration) {
	pickingStarter := student.Target == agents.TargetStarter1 ||
		student.Target == agents.TargetStarter2

	if pickingStarter {
		pickupTime := min(*remainingTime, student.RemainingStarterPickup)

		*remainingTime -= pickupTime
		student.RemainingStarterPickup -= pickupTime

		if student.RemainingStarterPickup > 0 {
			*remainingTime = 0
			return
		}

		before := len(student.Meal.Starters)
		collectDish(world, student)
		collected := len(student.Meal.Starters) - before

		counterIndex := 0
		if student.Target == agents.TargetStarter2 {
			counterIndex = 1
		}

		recordService(
			&world.Statistics.StarterCounters[counterIndex],
			student.QueueWait,
			world.Settings.ServiceTime*time.Duration(collected),
			collected,
		)

		if student.Target == agents.TargetStarter1 {
			student.Target = agents.TargetStarter2
		} else {
			student.Target = agents.TargetMain
		}

		student.Status = agents.Walking
		student.QueueWait = 0
		return
	}

	// Main-course service.
	busyTime := world.Cook.AdvanceService(*remainingTime)
	*remainingTime -= busyTime

	if world.Cook.ServiceTimeLeft > 0 {
		*remainingTime = 0
		return
	}

	collectDish(world, student)

	recordService(
		&world.Statistics.MainCounter,
		student.QueueWait,
		world.Cook.Staff.ServiceTime,
		1,
	)

	world.Cook.StudentID = 0
	tickInfo.foodCounterFreeAt = tickInfo.dt - *remainingTime

	student.Status = agents.Walking
	student.Target = agents.TargetCashier
	student.QueueWait = 0
}

// Handle payment and departure from the queue.
func advancePayment(
	world *environment.Environment,
	student *agents.StudentState,
	id int,
	tickInfo *queueTick,
	remainingTime *time.Duration,
) bool {
	busyTime := world.Cashier.AdvanceService(*remainingTime)
	*remainingTime -= busyTime

	// Payment is still in progress.
	if world.Cashier.ServiceTimeLeft > 0 {
		*remainingTime = 0
		return false
	}

	// Payment completed.
	world.Cashier.StudentID = 0
	tickInfo.paymentCounterFreeAt = tickInfo.dt - *remainingTime

	recordService(
		&world.Statistics.CashierCounter,
		student.QueueWait,
		world.Cashier.Staff.ServiceTime,
		0,
	)

	student.Status = agents.Walking
	student.Target = agents.TargetDining
	student.NextWaypoint = 0

	world.Queue.Remove(id)

	advanceStudentAfterPayment(world, student, *remainingTime)
	*remainingTime = 0

	return true
}

// Count the reserved starters belonging to the current counter of a certain student
func countPlannedStarters(world *environment.Environment, student *agents.StudentState) int {
	counterNumber := 1
	if student.Target == agents.TargetStarter2 {
		counterNumber = 2
	}

	count := 0

	for _, dishID := range student.PlannedStarterIDs {
		for _, dish := range world.Menu {
			if dish.ID == dishID && dish.StarterCounter == counterNumber {
				count++
				break
			}
		}
	}

	return count
}

// Return uncollected reserved starters to inventory.
func restoreReservedStarters(world *environment.Environment, student *agents.StudentState) {
	for _, dishID := range student.PlannedStarterIDs {
		for i := range world.Menu {
			if world.Menu[i].ID == dishID {
				world.Menu[i].Portions++
				break
			}
		}
	}

	student.PlannedStarterIDs = nil
}

func advanceStudentAfterPayment(world *environment.Environment, student *agents.StudentState, dt time.Duration) {
	diningAreaIndex := (student.Student.ID - 1) % len(world.DiningAreas)
	diningArea := world.DiningAreas[diningAreaIndex]
	remainingTime := dt

	for remainingTime > 0 {
		switch student.Status {
		case agents.Walking:
			switch student.Target {
			case agents.TargetDining:
				remainingTime = student.Walk(diningArea.Route, world.Settings.WalkingSpeed, remainingTime)
				if student.NextWaypoint < len(diningArea.Route) {
					return
				}
				student.Status = agents.Dining
				student.RemainingDining = world.Settings.DiningTime

			case agents.TargetTrash:
				remainingTime = student.Walk(diningArea.CleanupRoute, world.Settings.WalkingSpeed, remainingTime)
				if student.NextWaypoint < len(diningArea.CleanupRoute) {
					return
				}
				student.Target = agents.TargetExit
				student.NextWaypoint = 0

			case agents.TargetExit:
				student.Walk(world.ExitRoute, world.Settings.WalkingSpeed, remainingTime)
				if student.NextWaypoint == len(world.ExitRoute) {
					student.Status = agents.Finished
				}
				return

			default:
				return
			}

		case agents.Dining:
			timeSpentEating := min(remainingTime, student.RemainingDining)
			remainingTime -= timeSpentEating
			student.RemainingDining -= timeSpentEating
			if student.RemainingDining > 0 {
				return
			}
			student.Status = agents.Walking
			student.Target = agents.TargetTrash
			student.NextWaypoint = 0

		default:
			return
		}
	}
}

// Communicate with the student's goroutine, while the engine remains responsible for applying the decision.
func (engine *engine) requestStudentDecision(student *agents.StudentState, observation agents.StudentObservation) agents.StudentDecision {
	id := student.Student.ID

	if engine.StudentAgents == nil {
		engine.StudentAgents = make(map[int]*StudentAgent)
	}

	agent := engine.StudentAgents[id]

	if agent == nil {
		agent = startStudentAgent(student, &engine.agentWG)
		engine.StudentAgents[id] = agent
	}

	return agent.ask(observation)
}

func collectDish(world *environment.Environment, student *agents.StudentState) {
	// Collect the selected main course.
	if student.Target == agents.TargetMain {
		for i := range world.Menu {
			dish := &world.Menu[i]

			if dish.Course != agents.Main || dish.ID != student.OrderedDishID || dish.Portions <= 0 {
				continue
			}

			meal := *dish
			student.Meal.Main = &meal
			dish.Portions--
			return
		}
		return
	}

	// Collect planned starters from the current counter.
	counterNumber := 1
	if student.Target == agents.TargetStarter2 {
		counterNumber = 2
	}

	remaining := make([]string, 0, len(student.PlannedStarterIDs))

	for _, dishID := range student.PlannedStarterIDs {
		collected := false

		for i := range world.Menu {
			dish := &world.Menu[i]

			if dish.Course != agents.Starter ||
				dish.ID != dishID ||
				dish.StarterCounter != counterNumber {
				continue
			}

			meal := *dish
			student.Meal.Starters = append(student.Meal.Starters, meal)
			collected = true
			break
		}

		if !collected {
			remaining = append(remaining, dishID)
		}
	}

	// if the wanted dish is at conter 2 but now the student is at conter 1, remaining reserve it
	student.PlannedStarterIDs = remaining
}

func recordService(stats *environment.CounterStatistics, wait, service time.Duration, dishes int) {
	stats.StudentsServed++
	stats.DishesCollected += dishes
	stats.TotalWaitTime += wait
	stats.TotalServiceTime += service

	if wait > stats.MaxWaitTime {
		stats.MaxWaitTime = wait
	}
}

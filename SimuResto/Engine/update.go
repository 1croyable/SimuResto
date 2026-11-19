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

	starter1Position     float64
	starter2Position     float64
	mainPosition         float64
	checkoutForkPosition float64
	doorPosition         float64

	foodCounterFreeAt    time.Duration
	cookRestocking       bool
	paymentCounterFreeAt [environment.MaxCashiers]time.Duration

	dt time.Duration
}

func (engine *engine) advanceQueue(world *environment.Environment, dt time.Duration, cookRestocking bool) {
	if world.Queue == nil || dt <= 0 {
		return
	}

	// The common FIFO line ends at the checkout fork, not at either cashier.
	// Checkout2Route[0] is the fork point (e.g. 540, 520).
	if len(world.Checkout2Route) == 0 {
		return
	}
	fork := world.Checkout2Route[0]
	route := append([]agents.Point(nil), world.Route...)
	route = append(route, world.Starter2Route...)
	route = append(route, world.MainRoute...)
	route = append(route, fork)

	q := queueTick{
		route:       route,
		starter1End: len(world.Route),
		dt:          dt,
	}

	q.cookRestocking = cookRestocking
	q.doorPosition = routeLength(world.Route[:3])

	q.starter2End = q.starter1End + len(world.Starter2Route)
	q.mainEnd = q.starter2End + len(world.MainRoute)

	q.starter1Position = routeLength(route[:q.starter1End])
	q.starter2Position = routeLength(route[:q.starter2End])
	q.mainPosition = routeLength(route[:q.mainEnd])
	q.checkoutForkPosition = routeLength(route)

	personAheadPosition := q.checkoutForkPosition + world.Settings.QueueSpacing
	peopleAhead := 0

	// calculate one by one from the front
	for _, id := range world.Queue.IDs() {
		student := world.StudentsByID[id]
		remainingTime := min(dt, world.TimePassed-student.Student.ArrivalTime)
		leftQueue := false

		for remainingTime > 0 {
			switch student.Status {

			case agents.Walking, agents.WaitingForFood, agents.WaitingForPayment:
				if student.Target == agents.TargetCashier && student.CashierIndex >= 0 {
					leftQueue = engine.advanceAssignedCashier(world, student, id, &q, &remainingTime)
				} else {
					leftQueue = engine.advanceTowardCounter(world, student, id, &q, &remainingTime, personAheadPosition, peopleAhead)
				}

			case agents.Serving:
				advanceFoodService(world, student, &q, &remainingTime)

			case agents.Paying:
				leftQueue = engine.advancePayment(world, student, id, &q, &remainingTime)

			default:
				remainingTime = 0
			}
		}

		if !leftQueue {
			// A dispatched student leaves the common waiting line, but still
			// blocks the fork until sufficiently far along their branch.
			if student.Target == agents.TargetCashier && student.CashierIndex >= 0 {
				if student.PositionAlongRoute < q.checkoutForkPosition+world.Settings.QueueSpacing {
					personAheadPosition = student.PositionAlongRoute
					peopleAhead++
				}
			} else {
				personAheadPosition = student.PositionAlongRoute
				peopleAhead++
			}
		}
	}
}

// Find an open cashier without a customer in service or en route.
// This reservation is necessary: the student must walk from the fork
// to the chosen counter before the StaffAgent accepts them.
func findAvailableCashier(world *environment.Environment, tickInfo *queueTick) (int, time.Duration) {
	selected := -1
	var freeAt time.Duration

	for i := 0; i < len(world.Cashiers) && i < world.Settings.Cashiers; i++ {
		if world.Cashiers[i].StudentID != 0 {
			continue
		}

		reserved := false
		for _, id := range world.Queue.IDs() {
			other := world.StudentsByID[id]
			if other.Target == agents.TargetCashier && other.CashierIndex == i {
				reserved = true
				break
			}
		}
		if reserved {
			continue
		}

		if selected == -1 || tickInfo.paymentCounterFreeAt[i] < freeAt {
			selected = i
			freeAt = tickInfo.paymentCounterFreeAt[i]
		}
	}

	return selected, freeAt
}

// Follow the selected checkout branch, then start payment only after arrival.
func (engine *engine) advanceAssignedCashier(
	world *environment.Environment,
	student *agents.StudentState,
	id int,
	tickInfo *queueTick,
	remainingTime *time.Duration,
) bool {
	i := student.CashierIndex
	if i < 0 || i >= len(world.Cashiers) {
		*remainingTime = 0
		return false
	}

	branch := world.CheckoutRoute // Cashier 1: original lower route.
	if i == 1 {
		branch = world.Checkout2Route // Cashier 2: curved upper route.
	}
	if len(branch) == 0 {
		*remainingTime = 0
		return false
	}

	if student.NextWaypoint < len(branch) {
		before := *remainingTime
		*remainingTime = student.Walk(branch, world.Settings.WalkingSpeed, *remainingTime)
		walked := before - *remainingTime
		student.PositionAlongRoute += world.Settings.WalkingSpeed * walked.Seconds()

		if student.NextWaypoint < len(branch) {
			return false // Still walking towards the assigned counter.
		}
	}

	student.Status = agents.WaitingForPayment
	cashier := &world.Cashiers[i]

	// A cashier may become available partway through this tick.
	used := tickInfo.dt - *remainingTime
	if freeAt := tickInfo.paymentCounterFreeAt[i]; freeAt > used {
		wait := min(*remainingTime, freeAt-used)
		student.QueueWait += wait
		*remainingTime -= wait
		if tickInfo.dt-*remainingTime < freeAt {
			return false
		}
	}
	if cashier.StudentID != 0 && cashier.StudentID != id {
		student.QueueWait += *remainingTime
		*remainingTime = 0
		return false
	}

	if cashier.StudentID == 0 {
		decision := engine.requestStaffDecision(cashier, agents.StaffObservation{
			NextStudentID: id,
			QueueLength:   world.Queue.Len(),
			Elapsed:       0,
			Open:          true, // An already assigned customer can finish after closing.
		})
		if decision.AcceptedStudentID != id {
			student.QueueWait += *remainingTime
			*remainingTime = 0
			return false
		}
		cashier.StudentID = id
		cashier.ServiceTimeLeft = decision.RemainingTime
	}

	student.Status = agents.Paying
	return false // advanceQueue will call advancePayment in the same tick if time remains.
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

	var counter *agents.StaffState
	counterPosition := tickInfo.checkoutForkPosition
	counterFreeAt := time.Duration(0)
	cashierIndex := -1

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
	stoppingPosition := min(counterPosition, personAheadPosition-world.Settings.QueueSpacing)

	waitingForOpening := world.OperatingPhase != environment.OperatingOpen

	if waitingForOpening &&
		student.PositionAlongRoute <= tickInfo.doorPosition+0.0001 {

		stoppingPosition = min(stoppingPosition, tickInfo.doorPosition)
	}

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

	if waitingForOpening &&
		student.PositionAlongRoute <= tickInfo.doorPosition+0.0001 {

		// Waiting before opening is not counted as service queue waiting
		student.QueueWait = 0
		*remainingTime = 0
		return false
	}

	atCounter := student.PositionAlongRoute >= counterPosition-0.0001

	// Choose two starters before counter 1; do not consume stock until pickup.
	if student.Target == agents.TargetStarter1 &&
		student.PlannedStarterIDs == nil &&
		student.PositionAlongRoute >= tickInfo.starter1Position-20 {

		decision := engine.requestStudentDecision(
			student,
			agents.StudentObservation{
				ChooseStarters: true,
				Dishes:         world.Menu,
				Temperature:    world.Settings.ChoiceTemperature,
			},
		)

		student.PlannedStarterIDs = decision.StarterIDs

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

	var dishes []agents.Dish
	canStartService := false

	switch {
	case pickingStarter:
		canStartService = atCounter && plannedStartersInStock(world, student)

	case beforeFood:
		if atCounter && student.OrderedDishID == "" {
			for _, dish := range world.Menu {
				if dish.Course != agents.Main || (student.Student.Vegetarian && !dish.Vegetarian) {
					continue
				}
				dishes = append(dishes, dish)
			}
			if len(dishes) > 0 {
				choice := engine.requestStudentDecision(student, agents.StudentObservation{
					Dishes:      dishes,
					Temperature: world.Settings.ChoiceTemperature,
				})
				student.OrderedDishID = choice.DishID
			}
		}
		canStartService = atCounter && !tickInfo.cookRestocking &&
			counter.StudentID == 0 && student.OrderedDishID != "" &&
			world.HasStock(student.OrderedDishID, 1)

	default:
		// Select an available cashier for payment.
		if atCounter {
			cashierIndex, counterFreeAt = findAvailableCashier(world, tickInfo)
			canStartService = cashierIndex >= 0
		}
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
		abandonStudent(world, student)
		*remainingTime = 0
		return true
	}

	if !canStartService || (*remainingTime == 0 && student.Target != agents.TargetCashier) {
		switch {
		case student.Target == agents.TargetCashier:
			student.Status = agents.WaitingForPayment
		case beforeFood, pickingStarter:
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
		student.RemainingStarterPickup = world.Settings.ServiceTime

	case beforeFood:
		// Main course has already been chosen; stock was checked before service.
		student.Position = world.MainRoute[len(world.MainRoute)-1]
		student.NextWaypoint = tickInfo.mainEnd
		student.Status = agents.Serving

	default:
		student.CashierIndex = cashierIndex
		student.NextWaypoint = 0 // Waypoint index is now local to the branch.
		student.Status = agents.Walking
	}

	if beforeFood {
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

		counterIndex := 0
		counterNumber := 1
		if student.Target == agents.TargetStarter2 {
			counterIndex = 1
			counterNumber = 2
		}

		// Check whether this student has already collected another dish at the same counter.
		alreadyServed := false
		for _, dish := range student.Meal.Starters {
			if dish.StarterCounter == counterNumber {
				alreadyServed = true
				break
			}
		}

		before := len(student.Meal.Starters)
		collectDish(world, student)

		if len(student.Meal.Starters) == before {
			student.Status = agents.WaitingForFood
			return
		}

		// Record exactly one collected dish and one pickup duration.
		stats := &world.Statistics.StarterCounters[counterIndex]

		if !alreadyServed {
			stats.StudentsServed++
		}

		stats.DishesCollected++
		stats.TotalServiceTime += world.Settings.ServiceTime
		stats.TotalWaitTime += student.QueueWait

		if student.QueueWait > stats.MaxWaitTime {
			stats.MaxWaitTime = student.QueueWait
		}

		student.QueueWait = 0

		// Still another dish to collect at this counter?
		if countPlannedStarters(world, student) > 0 {
			student.Status = agents.WaitingForFood
			return
		}

		// All dishes at this counter have been collected.
		if student.Target == agents.TargetStarter1 {
			student.Target = agents.TargetStarter2
		} else {
			student.Target = agents.TargetMain
		}

		student.Status = agents.Walking
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
func (engine *engine) advancePayment(
	world *environment.Environment,
	student *agents.StudentState,
	id int,
	tickInfo *queueTick,
	remainingTime *time.Duration,
) bool {
	// Find the cashier currently serving this student.
	cashierIndex := -1

	for i := range world.Cashiers {
		if world.Cashiers[i].StudentID == id {
			cashierIndex = i
			break
		}
	}

	if cashierIndex < 0 {
		student.Status = agents.WaitingForPayment
		*remainingTime = 0
		return false
	}

	cashier := &world.Cashiers[cashierIndex]

	// Ask the cashier agent to advance the service.
	decision := engine.requestStaffDecision(
		cashier,
		agents.StaffObservation{
			Elapsed: *remainingTime,
			Open:    cashierIndex < world.Settings.Cashiers,
		},
	)

	// Apply the decision to Environment.
	cashier.ServiceTimeLeft = decision.RemainingTime
	*remainingTime -= decision.BusyTime

	// Payment is still in progress.
	if decision.CompletedStudentID != id {
		*remainingTime = 0
		return false
	}

	// Payment completed.
	cashier.StudentID = 0

	tickInfo.paymentCounterFreeAt[cashierIndex] =
		tickInfo.dt - *remainingTime

	recordService(
		&world.Statistics.CashierCounter,
		student.QueueWait,
		cashier.Staff.ServiceTime,
		0,
	)

	student.Status = agents.Walking
	student.Target = agents.TargetDining
	student.NextWaypoint = 0

	world.Queue.Remove(id)

	advanceStudentOutsideQueue(world, student, *remainingTime)
	*remainingTime = 0

	return true
}

func plannedStartersInStock(world *environment.Environment, student *agents.StudentState) bool {
	counterNumber := 1
	if student.Target == agents.TargetStarter2 {
		counterNumber = 2
	}

	for _, dishID := range student.PlannedStarterIDs {
		for _, dish := range world.Menu {
			if dish.ID == dishID &&
				dish.Course == agents.Starter &&
				dish.StarterCounter == counterNumber {
				return world.HasStock(dishID, 1)
			}
		}
	}

	return false
}

// Count the uncollected starters requested from the current counter.
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

// Forget pending starter choices; selecting a dish no longer reserves stock.
func restoreReservedStarters(_ *environment.Environment, student *agents.StudentState) {
	// Selection no longer reserves stock. Simply forget uncollected choices.
	student.PlannedStarterIDs = nil
}

func advanceStudentOutsideQueue(world *environment.Environment, student *agents.StudentState, dt time.Duration) {
	// if this student has given up
	if student.Status == agents.Abandoned && len(student.AbandonRoute) > 0 {

		student.Walk(student.AbandonRoute, world.Settings.WalkingSpeed, dt)

		if student.NextWaypoint >= len(student.AbandonRoute) {
			student.AbandonRoute = nil
		}

		return
	}

	diningAreaIndex := (student.Student.ID - 1) % len(world.DiningAreas)
	diningArea := world.DiningAreas[diningAreaIndex]
	remainingTime := dt

	for remainingTime > 0 {
		switch student.Status {
		case agents.Walking:
			switch student.Target {
			case agents.TargetDining:
				diningRoute := diningArea.Route
				if student.CashierIndex == 1 {
					// Continue forward from the upper cashier, without returning
					// to the lower checkout road.
					diningRoute = append([]agents.Point(nil), world.Cashier2ExitRoute...)
					diningRoute = append(diningRoute, diningArea.UpperCashierRoute...)
				}
				remainingTime = student.Walk(diningRoute, world.Settings.WalkingSpeed, remainingTime)
				if student.NextWaypoint < len(diningRoute) {
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

			if dish.Course != agents.Main || dish.ID != student.OrderedDishID {
				continue
			}

			if !world.TakeDish(dish.ID) {
				return
			}
			meal := *dish
			meal.Portions = 1
			student.Meal.Main = &meal
			return
		}
		return
	}

	// Collect one starter from the current counter.
	counterNumber := 1
	if student.Target == agents.TargetStarter2 {
		counterNumber = 2
	}

	for i, dishID := range student.PlannedStarterIDs {
		for j := range world.Menu {
			dish := &world.Menu[j]

			if dish.Course != agents.Starter ||
				dish.ID != dishID ||
				dish.StarterCounter != counterNumber {
				continue
			}

			if !world.TakeDish(dishID) {
				return
			}

			meal := *dish
			meal.Portions = 1

			student.Meal.Starters = append(
				student.Meal.Starters,
				meal,
			)

			// Remove only the dish that was just collected.
			student.PlannedStarterIDs = append(
				student.PlannedStarterIDs[:i],
				student.PlannedStarterIDs[i+1:]...,
			)

			return
		}
	}
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

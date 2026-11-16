package engine

import (
	"math"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func Step(world *environment.Environment, dt time.Duration) {
	if world.Paused || dt <= 0 {
		return
	}

	world.TimePassed += dt

	arrivedStudents := make(chan *agents.StudentState, len(world.Students))
	for studentIndex := range world.Students {
		student := &world.Students[studentIndex]
		if student.Status == agents.Scheduled && student.Student.ArrivalTime <= world.TimePassed {
			arrivedStudents <- student
		}
	}
	close(arrivedStudents)

	// admit arrived students while there is space in the queue
	admitStudents(world, arrivedStudents)

	// advance student state
	for studentIndex := range world.Students {
		student := &world.Students[studentIndex]
		advanceStudentAfterPayment(world, student, dt)
	}

	// students still in line move, collect food, pay, or leave the queue
	advanceQueue(world, dt)

	// fill places freed by payment or abandonment during this step
	admitStudents(world, arrivedStudents)
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

func advanceQueue(world *environment.Environment, dt time.Duration) {
	if world.Queue == nil {
		return
	}

	// total route of the queue, until the checkout counter
	route := append([]agents.Point(nil), world.Route...)
	route = append(route, world.Starter2Route...)
	route = append(route, world.MainRoute...)
	route = append(route, world.CheckoutRoute...)

	starter1End := len(world.Route)
	starter2End := starter1End + len(world.Starter2Route)
	mainEnd := starter2End + len(world.MainRoute)

	starter1Position := routeLength(route[:starter1End])
	starter2Position := routeLength(route[:starter2End])
	foodCounterPositionAlongRoute := routeLength(route[:mainEnd]) // start to food
	paymentCounterPositionAlongRoute := routeLength(route)        // start to checkout

	personAheadPositionAlongRoute := paymentCounterPositionAlongRoute + world.Settings.QueueSpacing // start to checkout + 1 space

	peopleAhead := 0
	var foodCounterFreeAt, paymentCounterFreeAt time.Duration

	// calculate one by one from the front
	for _, id := range world.Queue.IDs() {
		student := world.StudentsByID[id]
		leftQueue := false // give up waiting or finish paying => true
		remainingTime := min(dt, world.TimePassed-student.Student.ArrivalTime)

		for remainingTime > 0 {
			// update status according to remaining time
			switch student.Status {
			case agents.Walking, agents.WaitingForFood, agents.WaitingForPayment:
				// Select the counter, its position along the route, and when it becomes free.
				beforeFood := student.Target == agents.TargetMain
				pickingStarter := student.Target == agents.TargetStarter1 || student.Target == agents.TargetStarter2

				counter := &world.Cashier
				counterPositionAlongRoute := paymentCounterPositionAlongRoute
				counterFreeAt := paymentCounterFreeAt

				if beforeFood {
					counter = &world.Cook
					counterPositionAlongRoute = foodCounterPositionAlongRoute
					counterFreeAt = foodCounterFreeAt
				} else if student.Target == agents.TargetStarter1 {
					counterPositionAlongRoute = starter1Position
					counterFreeAt = 0
				} else if student.Target == agents.TargetStarter2 {
					counterPositionAlongRoute = starter2Position
					counterFreeAt = 0
				}

				// the furthest position that the current student is allowed to reach is either the corresponding counter or the next position
				stoppingPositionAlongRoute := min(counterPositionAlongRoute, personAheadPositionAlongRoute-world.Settings.QueueSpacing)
				distanceToStoppingPosition := max(0, stoppingPositionAlongRoute-student.PositionAlongRoute)

				// calculate walking time
				timeToStoppingPosition := time.Duration(math.Round(distanceToStoppingPosition / world.Settings.WalkingSpeed * float64(time.Second)))
				walkingTime := min(remainingTime, timeToStoppingPosition)

				// move student
				student.Walk(route, world.Settings.WalkingSpeed, walkingTime)
				student.PositionAlongRoute = min(stoppingPositionAlongRoute, student.PositionAlongRoute+world.Settings.WalkingSpeed*walkingTime.Seconds())
				remainingTime -= walkingTime

				atCounter := student.PositionAlongRoute >= counterPositionAlongRoute-0.0001
				var dishes []agents.DishOption
				canStartService := false

				if pickingStarter {
					// self-service: no staff required
					canStartService = atCounter
				} else if beforeFood {
					// food service
					if atCounter && counter.StudentID == 0 {
						for _, dish := range world.Menu {
							if dish.Portions > 0 {
								dishes = append(dishes, agents.DishOption{ID: dish.ID})
							}
						}
						canStartService = len(dishes) > 0
					}
				} else {
					// payment service
					canStartService = atCounter && counter.StudentID == 0
				}

				waitingTime := remainingTime
				if canStartService {
					timeAlreadyUsed := dt - remainingTime
					if counterFreeAt > timeAlreadyUsed {
						waitingTime = min(remainingTime, counterFreeAt-timeAlreadyUsed)
					} else {
						waitingTime = 0
					}
				}

				student.QueueWait += waitingTime

				remainingTime -= waitingTime

				decision := student.Decide(agents.StudentObservation{
					CanLeave:    student.Target != agents.TargetCashier,
					PeopleAhead: peopleAhead,
					Waiting:     student.QueueWait,
					Elapsed:     waitingTime,
				})

				if decision.Leave {
					student.Status = agents.Abandoned
					world.Queue.Remove(id)
					leftQueue = true
					remainingTime = 0
					break
				}
				if !canStartService || remainingTime == 0 {
					if student.Target == agents.TargetCashier {
						student.Status = agents.WaitingForPayment
					} else if beforeFood {
						student.Status = agents.WaitingForFood
					}
					break
				}

				// update all of the student data
				student.PositionAlongRoute = counterPositionAlongRoute

				if pickingStarter {
					if student.Target == agents.TargetStarter1 {
						student.Position = route[starter1End-1]
						student.NextWaypoint = starter1End
					} else {
						student.Position = route[starter2End-1]
						student.NextWaypoint = starter2End
					}
					student.Status = agents.Serving
					student.RemainingStarterPickup = world.Settings.ServiceTime

				} else if beforeFood {
					choice := student.Decide(agents.StudentObservation{Dishes: dishes})
					student.OrderedDishID = choice.DishID
					student.Position = world.MainRoute[len(world.MainRoute)-1]
					student.NextWaypoint = mainEnd
					student.Status = agents.Serving
				} else {
					student.Position = route[len(route)-1]
					student.NextWaypoint = len(route)
					student.Status = agents.Paying
				}

				if !pickingStarter {
					counter.StudentID = id
					counter.ServiceTimeLeft = counter.Staff.ServiceTime
				}

			case agents.Serving:
				// self-service starter pickup
				if student.Target == agents.TargetStarter1 || student.Target == agents.TargetStarter2 {
					pickupTime := min(remainingTime, student.RemainingStarterPickup)
					remainingTime -= pickupTime
					student.RemainingStarterPickup -= pickupTime

					if student.RemainingStarterPickup > 0 {
						break
					}

					if student.Target == agents.TargetStarter1 {
						student.Target = agents.TargetStarter2
					} else {
						student.Target = agents.TargetMain
					}
					student.Status = agents.Walking
					student.QueueWait = 0
					break
				}

				busyTime := world.Cook.AdvanceService(remainingTime)
				remainingTime -= busyTime
				if world.Cook.ServiceTimeLeft > 0 {
					break
				}
				for i := range world.Menu {
					dish := &world.Menu[i]
					if dish.ID == student.OrderedDishID {
						meal := *dish
						student.Meal.Main = &meal
						dish.Portions--
						world.RemainingPortions--
						break
					}
				}
				world.Cook.StudentID = 0
				foodCounterFreeAt = dt - remainingTime
				student.Status = agents.Walking
				student.Target = agents.TargetCashier
				student.QueueWait = 0

			case agents.Paying:
				busyTime := world.Cashier.AdvanceService(remainingTime)
				remainingTime -= busyTime
				if world.Cashier.ServiceTimeLeft > 0 {
					break
				}
				world.Cashier.StudentID = 0
				paymentCounterFreeAt = dt - remainingTime
				student.Status = agents.Walking
				student.Target = agents.TargetDining
				student.NextWaypoint = 0
				world.Queue.Remove(id)
				leftQueue = true
				advanceStudentAfterPayment(world, student, remainingTime)
				remainingTime = 0

			default:
				remainingTime = 0
			}
		}

		if !leftQueue {
			personAheadPositionAlongRoute = student.PositionAlongRoute
			peopleAhead++
		}
	}
}

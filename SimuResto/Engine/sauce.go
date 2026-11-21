package engine

import (
	"slices"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

// The line starts above the station and extends to its right.
func (e *engine) sauceQueuePosition(world *environment.Environment, index int) agents.Point {
	region := world.WaterAndSauceRegion
	entrance := e.dining.Point(e.dining.SauceEntrance)
	head := agents.Point{X: float64(region.X + region.Width/2), Y: entrance.Y}
	capacity := world.Sauce.Queue.Capacity()
	if capacity == 1 {
		return head
	}
	spacing := min(world.Settings.QueueSpacing, (entrance.X-head.X)/float64(capacity-1))
	return agents.Point{X: head.X + float64(index)*spacing, Y: head.Y}
}

func (e *engine) joinSauceQueue(world *environment.Environment, student *agents.StudentState) bool {
	station := &world.Sauce
	index := station.Queue.Len()
	if station.Portions == 0 || !station.Queue.Enqueue(student.Student.ID) {
		return false
	}
	slot := e.sauceQueuePosition(world, index)
	var approach []agents.Point
	var from int
	if student.SeatID != 0 {
		seat := e.dining.Seats[student.SeatID-1]
		from = seat.Node
		approach = []agents.Point{seat.Access, e.dining.Point(from)}
	} else {
		from = e.dining.CheckoutNodes[student.CashierIndex]
	}
	route := e.dining.AppendRoute(approach, e.dining.Search(from), e.dining.SauceEntrance)
	student.DiningRoute = append(route, slot)
	student.Target = agents.TargetSauce
	student.Status = agents.Walking
	student.NextWaypoint = 0
	student.RemainingSaucePickup = environment.SaucePickupTime
	return true
}

// The first student starts pickup after reaching the station and uses one portion on completion.
func (e *engine) advanceSaucePickup(world *environment.Environment, student *agents.StudentState, dt time.Duration) time.Duration {
	station := &world.Sauce
	index := slices.Index(station.Queue.IDs(), student.Student.ID)
	last := len(student.DiningRoute) - 1
	slot := e.sauceQueuePosition(world, index)
	if student.DiningRoute[last] != slot {
		student.DiningRoute[last] = slot
		student.NextWaypoint = min(student.NextWaypoint, last)
		student.Status = agents.Walking
	}

	dt = student.Walk(student.DiningRoute, world.Settings.WalkingSpeed, dt)
	if student.NextWaypoint < len(student.DiningRoute) {
		return 0
	}
	student.Status = agents.WaitingForSauce
	if index != 0 {
		return 0
	}

	// If the station turns over within a tick, start from the previous pickup's end.
	dt = min(dt, max(0, world.TimePassed-station.FreeAt))
	used := min(dt, student.RemainingSaucePickup)
	student.RemainingSaucePickup -= used
	if student.RemainingSaucePickup > 0 {
		return 0
	}

	station.FreeAt = world.TimePassed - (dt - used)
	station.Queue.Dequeue()
	station.Portions--
	e.leaveSauceQueue(world, student)
	if station.Portions == 0 {
		for _, id := range station.Queue.IDs() {
			station.Queue.Dequeue()
			e.leaveSauceQueue(world, world.StudentsByID[id])
		}
	}
	return dt - used
}

func (e *engine) leaveSauceQueue(world *environment.Environment, student *agents.StudentState) {
	last := len(student.DiningRoute) - 1
	var approach []agents.Point
	var from int
	if student.NextWaypoint < last {
		// The station ran out while the student was still approaching the queue.
		from = e.dining.SauceEntrance
		approach = student.DiningRoute[student.NextWaypoint:last]
	} else {
		from = e.dining.SauceExit
		exit := e.dining.Point(from)
		approach = []agents.Point{{X: student.Position.X, Y: exit.Y}, exit}
	}
	student.RemainingSaucePickup = 0
	if student.SeatID == 0 {
		e.requestSeat(world, student, from, approach)
		return
	}
	student.Status = agents.Walking
	student.NextWaypoint = 0
	student.DiningRoute = approach
	e.routeToSeat(student, e.dining.Search(from))
}

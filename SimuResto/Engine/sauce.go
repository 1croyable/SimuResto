package engine

import (
	"slices"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func diningAreaFor(world *environment.Environment, student *agents.StudentState) environment.DiningArea {
	return world.DiningAreas[(student.Student.ID-1)%len(world.DiningAreas)]
}

// The line starts above the station and extends to its right.
func sauceQueuePosition(world *environment.Environment, index int) agents.Point {
	region := world.WaterAndSauceRegion
	head := agents.Point{X: float64(region.X + region.Width/2), Y: float64(region.Y) - 12}
	aisleX := world.DiningAreas[0].Route[0].X
	spacing := min(world.Settings.QueueSpacing, (aisleX-head.X)/float64(environment.SauceCapacity-1))
	return agents.Point{X: head.X + float64(index)*spacing, Y: head.Y}
}

func joinSauceQueue(world *environment.Environment, student *agents.StudentState) bool {
	station := &world.Sauce
	index := station.Queue.Len()
	if station.Portions == 0 || !station.Queue.Enqueue(student.Student.ID) {
		return false
	}
	slot := sauceQueuePosition(world, index)
	aisleX := diningAreaFor(world, student).Route[0].X
	student.SauceRoute = []agents.Point{{X: aisleX, Y: student.Position.Y}, {X: aisleX, Y: slot.Y}, slot}
	student.Target = agents.TargetSauce
	student.Status = agents.Walking
	student.NextWaypoint = 0
	student.RemainingSaucePickup = environment.SaucePickupTime
	return true
}

// The first student starts pickup after reaching the station and uses one portion on completion.
func advanceSaucePickup(world *environment.Environment, student *agents.StudentState, dt time.Duration) time.Duration {
	station := &world.Sauce
	index := slices.Index(station.Queue.IDs(), student.Student.ID)
	last := len(student.SauceRoute) - 1
	slot := sauceQueuePosition(world, index)
	if student.SauceRoute[last] != slot {
		student.SauceRoute[last] = slot
		student.NextWaypoint = min(student.NextWaypoint, last)
		student.Status = agents.Walking
	}

	dt = student.Walk(student.SauceRoute, world.Settings.WalkingSpeed, dt)
	if student.NextWaypoint < len(student.SauceRoute) {
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
	leaveSauceQueue(world, student)
	if station.Portions == 0 {
		for _, id := range station.Queue.IDs() {
			station.Queue.Dequeue()
			leaveSauceQueue(world, world.StudentsByID[id])
		}
	}
	return dt - used
}

func leaveSauceQueue(world *environment.Environment, student *agents.StudentState) {
	area := diningAreaFor(world, student)
	aisleX := area.Route[0].X
	student.SauceRoute = student.SauceRoute[:0]
	queueY := sauceQueuePosition(world, 0).Y
	if student.Position.Y == queueY && student.Position.X < aisleX {
		// Leave via the lane beside the queue.
		student.SauceRoute = append(student.SauceRoute, agents.Point{X: student.Position.X, Y: queueY - 16})
		queueY -= 16
	} else {
		queueY = student.Position.Y
	}
	student.SauceRoute = append(student.SauceRoute, agents.Point{X: aisleX, Y: queueY})
	student.SauceRoute = append(student.SauceRoute, area.Route[1:]...)
	student.Target = agents.TargetDining
	student.Status = agents.Walking
	student.NextWaypoint = 0
	student.RemainingSaucePickup = 0
}

package engine

import (
	"math"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func routeLength(route []agents.Point) float64 {
	var totalDistance float64
	for i := 1; i < len(route); i++ {
		previousPoint, currentPoint := route[i-1], route[i]
		totalDistance += math.Hypot(currentPoint.X-previousPoint.X, currentPoint.Y-previousPoint.Y)
	}
	return totalDistance
}

func (e *engine) admitStudents(world *environment.Environment, arrivals []*agents.StudentState) {
	if world.Queue == nil {
		return
	}

	for _, student := range arrivals {
		if world.Queue.Len() >= world.Queue.Capacity() {
			panic("arrival queue exceeds planned capacity")
		}

		if e.startCompanionWalk(world, student) {
			continue
		}
		enqueueAtTail(world, student)
	}
}

func queueTailPosition(world *environment.Environment, ids []int) float64 {
	if len(ids) == 0 {
		return 0
	}
	last := world.StudentsByID[ids[len(ids)-1]]
	return min(0, last.PositionAlongRoute-world.Settings.QueueSpacing)
}

func enqueueAtTail(world *environment.Environment, student *agents.StudentState) {
	position := queueTailPosition(world, world.Queue.IDs())
	student.PositionAlongRoute = position
	student.Position = agents.Point{X: world.Starter1Route[0].X - position, Y: world.Starter1Route[0].Y}
	student.NextWaypoint = 0
	student.Status = agents.Walking
	student.Target = agents.TargetStarter1
	world.Queue.Enqueue(student.Student.ID)
}

// Remove students who are still on the outdoor part of the route at closing.
func (e *engine) removeOutsideStudents(world *environment.Environment) {
	if len(world.Starter1Route) <= environment.Starter1DoorIndex {
		return
	}

	doorPosition := routeLength(world.Starter1Route[:environment.Starter1DoorIndex+1])

	for _, id := range world.Queue.IDs() {
		student := world.StudentsByID[id]

		if (student.Target == agents.TargetStarter1 || student.Target == agents.TargetQueueSlot) && student.PositionAlongRoute <= doorPosition+0.0001 {
			abandonStudent(world, student)
		}
	}
	for _, student := range e.sideLaneCompanions {
		abandonStudent(world, student)
	}
	clear(e.sideLaneCompanions)
	e.sideLaneCompanions = e.sideLaneCompanions[:0]
}

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

	for len(arrivals) > 0 {
		student := arrivals[0]
		arrivals = arrivals[1:]
		if world.Queue.Len() >= world.Queue.Capacity() {
			panic("arrival queue exceeds planned capacity")
		}

		if student.Student.Companion.Role == agents.TogetherLeader {
			companionCount := 0
			for _, next := range arrivals {
				if next.Student.Companion.LeaderID != student.Student.ID {
					break
				}
				companionCount++
			}
			enqueueTogetherAtTail(world, student, arrivals[:companionCount])
			arrivals = arrivals[companionCount:]
			continue
		}

		if e.startCompanionWalk(world, student) {
			continue
		}
		enqueueAtTail(world, student)
	}
}

func enqueueTogetherAtTail(world *environment.Environment, leader *agents.StudentState, companions []*agents.StudentState) {
	if world.Queue.Len()+1+len(companions) > world.Queue.Capacity() {
		panic("arrival queue exceeds planned capacity")
	}
	spacing := world.Settings.QueueSpacing
	// Place the last companion at the entrance when the queue leaves enough room.
	position := float64(len(companions)) * spacing
	ids := world.Queue.IDs()
	if len(ids) > 0 {
		tail := world.StudentsByID[ids[len(ids)-1]]
		position = min(position, tail.PositionAlongRoute-spacing)
	}
	enqueueAtPosition(world, leader, position)
	for _, companion := range companions {
		position -= spacing
		enqueueAtPosition(world, companion, position)
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
	enqueueAtPosition(world, student, queueTailPosition(world, world.Queue.IDs()))
}

func enqueueAtPosition(world *environment.Environment, student *agents.StudentState, position float64) {
	student.PositionAlongRoute = position
	student.Position, student.NextWaypoint = queueJumpRoutePoint(world, position)
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

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

		student.PositionAlongRoute = 0

		ids := world.Queue.IDs()
		if len(ids) > 0 {
			last := world.StudentsByID[ids[len(ids)-1]]
			student.PositionAlongRoute = min(
				0,
				last.PositionAlongRoute-world.Settings.QueueSpacing,
			)
		}

		student.Position = agents.Point{
			X: world.Route[0].X - student.PositionAlongRoute,
			Y: world.Route[0].Y,
		}

		student.NextWaypoint = 0
		student.Status = agents.Walking
		student.Target = agents.TargetStarter1

		world.Queue.Enqueue(student.Student.ID)
	}
}

// called when the door close but some students are still outside the door
func (e *engine) removeOutsideStudents(world *environment.Environment) {
	if len(world.Route) < 3 {
		return
	}

	doorPosition := routeLength(world.Route[:3])

	for _, id := range world.Queue.IDs() {
		student := world.StudentsByID[id]

		if student.Target == agents.TargetStarter1 && student.PositionAlongRoute <= doorPosition+0.0001 {
			abandonStudent(world, student)
		}
	}
}

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

func admitStudents(world *environment.Environment, arrivals <-chan *agents.StudentState) {
	if world.Queue == nil {
		return
	}
	for world.Queue.Len() < world.Queue.Capacity() {
		student, ok := <-arrivals
		if !ok {
			return
		}
		student.PositionAlongRoute = 0
		ids := world.Queue.IDs()
		if len(ids) > 0 {
			lastStudent := world.StudentsByID[ids[len(ids)-1]]
			student.PositionAlongRoute = min(0, lastStudent.PositionAlongRoute-world.Settings.QueueSpacing)
		}
		student.Position = agents.Point{
			X: world.Route[0].X - student.PositionAlongRoute,
			Y: world.Route[0].Y,
		}
		student.NextWaypoint = 0
		student.Status = agents.WaitingForFood
		world.Queue.Enqueue(student.Student.ID)
	}
}

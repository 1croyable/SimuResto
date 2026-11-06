package agents

import (
	"math"
	"time"
)

type StudentData struct {
	ID             int
	ArrivalTime    time.Duration
	StartersWanted int
	DessertsWanted int
}

// Walk follows the route at walkingSpeed map units per second and returns the unused time.
func (student *StudentState) Walk(route []Point, walkingSpeed float64, dt time.Duration) time.Duration {
	for student.NextWaypoint < len(route) {
		target := route[student.NextWaypoint]
		dx, dy := target.X-student.Position.X, target.Y-student.Position.Y
		distance := math.Hypot(dx, dy)
		// total time needed to reach the next route point
		travelTime := time.Duration(math.Round(distance / walkingSpeed * float64(time.Second)))

		if travelTime > dt {
			// fraction is the share of the remaining segment covered this step, used to calculate the new position
			fraction := walkingSpeed * dt.Seconds() / distance

			// Move a part of the way toward the next point
			student.Position.X += dx * fraction
			student.Position.Y += dy * fraction
			return 0
		}

		student.Position = target
		student.NextWaypoint++
		dt -= travelTime
	}

	return dt
}

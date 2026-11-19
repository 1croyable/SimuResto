package engine

import (
	"math"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func buildAbandonRoute(world *environment.Environment, position agents.Point) []agents.Point {
	path := append([]agents.Point(nil), world.Route...)
	path = append(path, world.Starter2Route...)
	path = append(path, world.MainRoute...)

	if len(world.Checkout2Route) > 0 {
		path = append(path, world.Checkout2Route[0])
	}
	if len(path) < 2 {
		return nil
	}

	// Offset the incoming route by one queue spacing.
	gap := world.Settings.QueueSpacing
	normals := make([]agents.Point, len(path)-1)

	for i := range normals {
		dx := path[i+1].X - path[i].X
		dy := path[i+1].Y - path[i].Y
		length := math.Hypot(dx, dy)
		if length == 0 {
			return nil
		}
		normals[i] = agents.Point{X: dy / length, Y: -dx / length}
	}

	shifted := make([]agents.Point, len(path))
	for i := range path {
		n := normals[min(i, len(normals)-1)]

		if i > 0 && i < len(path)-1 {
			a, b := normals[i-1], normals[i]
			d := 1 + a.X*b.X + a.Y*b.Y
			if d > 0.0001 {
				n = agents.Point{
					X: (a.X + b.X) / d,
					Y: (a.Y + b.Y) / d,
				}
			}
		}

		shifted[i] = agents.Point{
			X: path[i].X + gap*n.X,
			Y: path[i].Y + gap*n.Y,
		}
	}

	// Find the closest segment of the original entrance route.
	bestSegment := 0
	bestFraction := 0.0
	bestDistance := math.Inf(1)

	for i := 0; i < len(path)-1; i++ {
		a, b := path[i], path[i+1]
		dx, dy := b.X-a.X, b.Y-a.Y
		lengthSquared := dx*dx + dy*dy

		t := ((position.X-a.X)*dx + (position.Y-a.Y)*dy) / lengthSquared
		t = max(0, min(1, t))

		x, y := a.X+t*dx, a.Y+t*dy
		distance := math.Hypot(position.X-x, position.Y-y)

		if distance < bestDistance {
			bestDistance = distance
			bestSegment = i
			bestFraction = t
		}
	}

	// First move sideways to the parallel lane.
	a, b := shifted[bestSegment], shifted[bestSegment+1]
	start := agents.Point{
		X: a.X + bestFraction*(b.X-a.X),
		Y: a.Y + bestFraction*(b.Y-a.Y),
	}

	route := []agents.Point{start}

	// Travel backwards along the parallel lane.
	for i := bestSegment; i >= 0; i-- {
		route = append(route, shifted[i])
	}

	// Leave the visible screen through the entrance.
	route = append(route, agents.Point{
		X: float64(environment.Width) + 20,
		Y: shifted[0].Y,
	})

	return route
}

func abandonStudent(world *environment.Environment, student *agents.StudentState) {
	restoreReservedStarters(world, student)

	student.AbandonRoute = buildAbandonRoute(world, student.Position)
	student.NextWaypoint = 0
	student.Status = agents.Abandoned

	world.Queue.Remove(student.Student.ID)
}

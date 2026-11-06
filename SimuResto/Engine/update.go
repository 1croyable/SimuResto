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

	start := world.TimePassed
	world.TimePassed += dt

	if world.TimePassed <= world.Student.Student.ArrivalTime {
		return
	}

	if start < world.Student.Student.ArrivalTime {
		dt = world.TimePassed - world.Student.Student.ArrivalTime
	}

	student := &world.Student
	for {
		switch student.Status {
		case agents.Walking:
			dt = walk(world, world.Route, dt)
			if student.NextWaypoint < len(world.Route) {
				return
			}
			student.Status = agents.Serving
			world.Cook.RemainingService = world.Cook.Staff.ServiceTime
		case agents.Serving:
			if world.RemainingPortions <= 0 {
				return
			}
			elapsed := min(dt, world.Cook.RemainingService)
			dt -= elapsed
			world.Cook.RemainingService -= elapsed
			if world.Cook.RemainingService > 0 {
				return
			}
			dish := world.Dish
			student.Meal.Main = &dish
			world.RemainingPortions--
			student.Status = agents.WalkingToCashier
			student.NextWaypoint = 0
		case agents.WalkingToCashier:
			dt = walk(world, world.CheckoutRoute, dt)
			if student.NextWaypoint < len(world.CheckoutRoute) {
				return
			}
			student.Status = agents.Paying
			world.Cashier.RemainingService = world.Cashier.Staff.ServiceTime
		case agents.Paying:
			elapsed := min(dt, world.Cashier.RemainingService)
			dt -= elapsed
			world.Cashier.RemainingService -= elapsed
			if world.Cashier.RemainingService > 0 {
				return
			}
			student.Status = agents.WalkingToDining
			student.NextWaypoint = 0
		case agents.WalkingToDining:
			area := diningArea(world)
			dt = walk(world, area.Route, dt)
			if student.NextWaypoint < len(area.Route) {
				return
			}
			student.Status = agents.Dining
			student.RemainingDining = world.Settings.DiningTime
		case agents.Dining:
			elapsed := min(dt, student.RemainingDining)
			dt -= elapsed
			student.RemainingDining -= elapsed
			if student.RemainingDining > 0 {
				return
			}
			student.Status = agents.WalkingToTrash
			student.NextWaypoint = 0
		case agents.WalkingToTrash:
			area := diningArea(world)
			dt = walk(world, area.CleanupRoute, dt)
			if student.NextWaypoint < len(area.CleanupRoute) {
				return
			}
			student.Status = agents.WalkingToExit
			student.NextWaypoint = 0
		case agents.WalkingToExit:
			walk(world, world.ExitRoute, dt)
			if student.NextWaypoint == len(world.ExitRoute) {
				student.Status = agents.Finished
			}
			return
		default:
			return
		}
	}
}

func diningArea(world *environment.Environment) environment.DiningArea {
	index := (max(1, world.Student.Student.ID) - 1) % len(world.DiningAreas)
	return world.DiningAreas[index]
}

// walk uses part or all of dt for movement and returns the time left.
func walk(world *environment.Environment, route []agents.Point, dt time.Duration) time.Duration {
	student := &world.Student

	for student.NextWaypoint < len(route) {
		target := route[student.NextWaypoint]
		dx, dy := target.X-student.Position.X, target.Y-student.Position.Y
		distance := math.Hypot(dx, dy)
		// Time needed to reach the next route point
		travelTime := time.Duration(math.Round(distance / world.Settings.WalkingSpeed * float64(time.Second)))

		if travelTime > dt {
			// fraction is the share of the remaining segment covered this step, not a duration.
			fraction := world.Settings.WalkingSpeed * dt.Seconds() / distance
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

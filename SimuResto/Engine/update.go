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

	if world.Student.Status == agents.Walking {
		// student walking
		dt = walk(world, dt)

		// switch to serving if the student has reached the counter
		if world.Student.NextWaypoint == len(world.Route) {
			world.Student.Status = agents.Serving
			world.Cook.RemainingService = world.Cook.Staff.ServiceTime
		}
	}

	if world.Student.Status == agents.Serving && world.RemainingPortions > 0 {
		// student being served
		world.Cook.RemainingService -= min(dt, world.Cook.RemainingService)

		// transfer the dish to the student
		if world.Cook.RemainingService == 0 {
			dish := world.Dish
			world.Student.Meal.Main = &dish
			world.RemainingPortions--
			world.Student.Status = agents.Served
		}
	}
}

func walk(world *environment.Environment, dt time.Duration) time.Duration {
	student := &world.Student

	for student.NextWaypoint < len(world.Route) {
		target := world.Route[student.NextWaypoint]
		dx, dy := target.X-student.Position.X, target.Y-student.Position.Y
		distance := math.Hypot(dx, dy)
		travelTime := time.Duration(
			math.Round(distance / world.Settings.WalkingSpeed * float64(time.Second))
		)

		if travelTime > dt {
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

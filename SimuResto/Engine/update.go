package engine

import (
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

	studentState := &world.Student
	for {
		switch studentState.Status {
		case agents.Walking:
			dt = studentState.Walk(world.Route, world.Settings.WalkingSpeed, dt)
			if studentState.NextWaypoint < len(world.Route) {
				return
			}
			studentState.Status = agents.Serving
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
			studentState.Meal.Main = &dish
			world.RemainingPortions--
			studentState.Status = agents.WalkingToCashier
			studentState.NextWaypoint = 0
		case agents.WalkingToCashier:
			dt = studentState.Walk(world.CheckoutRoute, world.Settings.WalkingSpeed, dt)
			if studentState.NextWaypoint < len(world.CheckoutRoute) {
				return
			}
			studentState.Status = agents.Paying
			world.Cashier.RemainingService = world.Cashier.Staff.ServiceTime
		case agents.Paying:
			elapsed := min(dt, world.Cashier.RemainingService)
			dt -= elapsed
			world.Cashier.RemainingService -= elapsed
			if world.Cashier.RemainingService > 0 {
				return
			}
			studentState.Status = agents.WalkingToDining
			studentState.NextWaypoint = 0
		case agents.WalkingToDining:
			area := diningArea(world)
			dt = studentState.Walk(area.Route, world.Settings.WalkingSpeed, dt)
			if studentState.NextWaypoint < len(area.Route) {
				return
			}
			studentState.Status = agents.Dining
			studentState.RemainingDining = world.Settings.DiningTime
		case agents.Dining:
			elapsed := min(dt, studentState.RemainingDining)
			dt -= elapsed
			studentState.RemainingDining -= elapsed
			if studentState.RemainingDining > 0 {
				return
			}
			studentState.Status = agents.WalkingToTrash
			studentState.NextWaypoint = 0
		case agents.WalkingToTrash:
			area := diningArea(world)
			dt = studentState.Walk(area.CleanupRoute, world.Settings.WalkingSpeed, dt)
			if studentState.NextWaypoint < len(area.CleanupRoute) {
				return
			}
			studentState.Status = agents.WalkingToExit
			studentState.NextWaypoint = 0
		case agents.WalkingToExit:
			studentState.Walk(world.ExitRoute, world.Settings.WalkingSpeed, dt)
			if studentState.NextWaypoint == len(world.ExitRoute) {
				studentState.Status = agents.Finished
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

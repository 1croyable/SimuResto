package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

// Engine applies the decision to Environment. Return true means the cook was restocking, so it may not begin serving a main course until the next tick.
func (e *engine) advanceRestocking(world *environment.Environment, dt time.Duration) bool {
	// Finish current main service
	if world.Cook.StudentID != 0 {
		return false
	}

	// create decision
	obs := agents.RestockObservation{Duration: world.Settings.RestockDuration}
	if world.Restock.Active {
		obs.Active = true
		obs.CounterIndex = world.Restock.CounterIndex
		obs.DishID = world.Restock.DishID
		obs.RemainingTime = world.Restock.Remaining
	} else {
		obs.Options = world.RefillOptions()
	}

	decision := e.requestStaffDecision(&world.Cook, agents.StaffObservation{
		Elapsed: dt,
		Restock: &obs,
	})

	// Record the time actually spent restocking.
	world.Statistics.CookRestockTime += decision.BusyTime

	if decision.RestockDishID == "" {
		return false
	}

	world.Restock = environment.RestockState{
		Active:       true,
		CounterIndex: decision.RestockCounter,
		DishID:       decision.RestockDishID,
		Remaining:    decision.RemainingTime,
	}

	if decision.RestockCompleted {
		if !world.RefillDish(
			decision.RestockCounter,
			decision.RestockDishID,
		) {
			panic("refill inventory invariant violated")
		}

		world.Statistics.RestockBatches++

		if world.AtMaxStock(decision.RestockDishID) {
			// All batches finished. The cook can return to main service.
			world.Restock = environment.RestockState{}
		} else {
			// Keep the same job and begin the next batch on the next tick.
			world.Restock.Remaining = 0
		}
	}
	return true
}

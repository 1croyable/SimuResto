package agents

import "time"

func (staff *StaffState) decideRestock(obs RestockObservation, elapsed time.Duration) StaffDecision {
	decision := StaffDecision{}
	if !obs.Active && staff.StudentID != 0 {
		return decision // Finish serving the main course before taking a refill job.
	}

	remaining := obs.RemainingTime
	if !obs.Active {
		if len(obs.Options) == 0 || obs.Duration <= 0 {
			return decision
		}
		// Empty dishes are ordered by counter and menu.
		selected := obs.Options[0]
		decision.RestockDishID = selected.DishID
		decision.RestockCounter = selected.CounterIndex
		remaining = obs.Duration
	} else {
		decision.RestockDishID = obs.DishID
		decision.RestockCounter = obs.CounterIndex

		// The previous batch finished; start another fixed-duration batch.
		if remaining <= 0 {
			remaining = obs.Duration
		}
	}

	if elapsed < 0 {
		elapsed = 0
	}
	decision.BusyTime = min(elapsed, remaining)
	decision.RemainingTime = remaining - decision.BusyTime
	decision.RestockCompleted = decision.RemainingTime <= 0
	return decision
}

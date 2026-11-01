package simulation

import "time"

func (e *engine) startSelfService() {
	for _, id := range e.queues[SelfService].IDs() {
		student := e.byID[id]
		if student.started {
			continue
		}
		student.started = true
		e.emit(SelfServiceStart, id, 0, SelfService, SelfService)
		student.missingSides = e.collectSides(student)
		var dishes []string
		for _, course := range [][]Dish{student.result.Meal.Starters, student.result.Meal.Desserts} {
			for _, dish := range course {
				dishes = append(dishes, dish.ID)
			}
		}
		e.emit(FoodCollected, id, 0, SelfService, SelfService, dishes...)
		if student.missingSides > 0 {
			e.emit(FoodUnavailable, id, 0, SelfService, SelfService)
		}
	}
}

// collectSides 先按期望组合领取，再用另一类补足；只对最终缺少的份数扣分。
func (e *engine) collectSides(student *studentState) int {
	profile, meal := student.result.Student, &student.result.Meal
	wanted := min(profile.StartersWanted+profile.DessertsWanted, e.config.MaxSelfServiceItems,
		e.config.MaxStartersPerStudent+e.config.MaxDessertsPerStudent)
	remaining := wanted
	take := func(target *[]Dish, choices []string, count, limit int) {
		dishes := e.stock.takeSides(choices, min(count, limit-len(*target), remaining))
		*target = append(*target, dishes...)
		remaining -= len(dishes)
	}
	take(&meal.Starters, student.plan.starterChoices, profile.StartersWanted, e.config.MaxStartersPerStudent)
	take(&meal.Desserts, student.plan.dessertChoices, profile.DessertsWanted, e.config.MaxDessertsPerStudent)
	take(&meal.Starters, student.plan.starterChoices, remaining, e.config.MaxStartersPerStudent)
	take(&meal.Desserts, student.plan.dessertChoices, remaining, e.config.MaxDessertsPerStudent)
	return remaining
}

func (e *engine) advanceSelfService(step time.Duration) {
	for _, id := range e.queues[SelfService].IDs() {
		student := e.byID[id]
		if student.started && !student.ready {
			student.elapsed += step
			if student.elapsed >= e.config.SelfServiceTime {
				student.ready = true
			}
		}
	}
}

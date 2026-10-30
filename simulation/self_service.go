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
		wantStarters := min(student.result.Student.StartersWanted,
			e.config.MaxStartersPerStudent,
			e.config.MaxSelfServiceItems)

		student.result.Meal.Starters = e.stock.takeSides(student.plan.starterChoices, wantStarters)
		wantDesserts := min(student.result.Student.DessertsWanted, e.config.MaxDessertsPerStudent,
			e.config.MaxSelfServiceItems-len(student.result.Meal.Starters))
		student.result.Meal.Desserts = e.stock.takeSides(student.plan.dessertChoices, wantDesserts)
		missing := wantStarters + wantDesserts - len(student.result.Meal.Starters) - len(student.result.Meal.Desserts)
		var dishes []string
		var bonus float32
		for _, course := range [][]Dish{student.result.Meal.Starters, student.result.Meal.Desserts} {
			if len(course) > 0 {
				bonus += e.config.CourseVarietyBonus
			}
			for _, dish := range course {
				bonus += dish.SatisfactionGain
				dishes = append(dishes, dish.ID)
			}
		}
		e.adjustSatisfaction(student, bonus-float32(missing)*e.config.MissingSidePenalty)
		e.emit(FoodCollected, id, 0, SelfService, SelfService, dishes...)
		if missing > 0 {
			e.emit(FoodUnavailable, id, 0, SelfService, SelfService)
		}
	}
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

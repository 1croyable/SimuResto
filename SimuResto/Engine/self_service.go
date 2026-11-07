package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

// 选前菜，甜点
func (e *engine) startSelfService() {
	for _, id := range e.queues[agents.SelfService].IDs() {
		student := e.ID2Student[id]
		if student.started {
			continue
		}
		student.started = true
		e.emit(agents.SelfServiceStart, id, 0, agents.SelfService, agents.SelfService)
		student.missingSides = e.collectSelfServiceItems(student)
		var dishes []string
		for _, course := range [][]agents.Dish{student.result.Meal.Starters, student.result.Meal.Desserts} {
			for _, dish := range course {
				dishes = append(dishes, dish.ID)
			}
		}
		e.emit(agents.FoodCollected, id, 0, agents.SelfService, agents.SelfService, dishes...)
		if student.missingSides > 0 {
			e.emit(agents.FoodUnavailable, id, 0, agents.SelfService, agents.SelfService)
		}
	}
}

// collectSelfServiceItems 先按期望组合领取，再用另一类补足；只对最终缺少的份数扣分。
func (e *engine) collectSelfServiceItems(student *studentState) int {
	profile, meal := student.result.Student, &student.result.Meal
	wanted := min(profile.StartersWanted+profile.DessertsWanted, e.config.MaxSelfServiceItems,
		e.config.MaxStartersPerStudent+e.config.MaxDessertsPerStudent)
	remaining := wanted
	take := func(target *[]agents.Dish, course agents.Course, count, limit int) {
		dishes := e.chooseAndTakeDishes(student, course, min(count, limit-len(*target), remaining))
		*target = append(*target, dishes...)
		remaining -= len(dishes)
	}
	take(&meal.Starters, agents.Starter, profile.StartersWanted, e.config.MaxStartersPerStudent)
	take(&meal.Desserts, agents.Dessert, profile.DessertsWanted, e.config.MaxDessertsPerStudent)
	take(&meal.Starters, agents.Starter, remaining, e.config.MaxStartersPerStudent)
	take(&meal.Desserts, agents.Dessert, remaining, e.config.MaxDessertsPerStudent)
	return remaining
}

// 每拿一份就重新读取库存、抽选并扣除，允许再次抽中尚有库存的同一道菜。
func (e *engine) chooseAndTakeDishes(student *studentState, course agents.Course, count int) []agents.Dish {
	// 当前学生完成本次领取后才释放库存锁，后续学生再读取最新库存。
	e.stock.mu.Lock()
	defer e.stock.mu.Unlock()
	var dishes []agents.Dish
	for range count {
		id := e.chooseDish(student, course)
		if id == "" {
			break
		}
		dishes = append(dishes, e.stock.takeSide(id))
	}
	return dishes
}

func (e *engine) stepSelfService(step time.Duration) {
	for _, id := range e.queues[agents.SelfService].IDs() {
		student := e.ID2Student[id]
		if student.started && !student.ready {
			student.elapsed += step
			if student.elapsed >= e.config.SelfServiceTime {
				student.ready = true
			}
		}
	}
}

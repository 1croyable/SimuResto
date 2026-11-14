package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

// 满意度使用 float32；时间使用 time.Duration。
func (e *engine) adjustSatisfaction(student *studentState, amount float32) {
	student.result.Satisfaction = min(100, max(0, student.result.Satisfaction+amount))
}

// dishScoreForStudent 计算个人评分，不修改菜单或库存中的原始评分。
// 荤素倾向用于主菜；前菜和甜点不会因不含肉而被肉食倾向学生扣分。
func dishScoreForStudent(dish agents.Dish, preference agents.Preference) float32 {
	score := dish.SatisfactionScore
	if dish.Course == agents.Main && ((preference == agents.Vegetarian) != dish.Vegetarian) {
		score -= environment.PreferenceMismatchPenalty
	}
	return max(0, score)
}

// mealSatisfactionChange 按实际领取的每一份菜的个人评分加权，再以 50 分为中性点。
// 两份甜点各使用一次甜点权重；没拿的类别不计入分母。
func mealSatisfactionChange(meal agents.Meal, preference agents.Preference) float32 {
	// 主菜、前菜、甜点固定按 2:1:1 计算。
	var weightedChange, totalWeight float64
	add := func(dish agents.Dish, weight float32) {
		weightedChange += float64(dishScoreForStudent(dish, preference)-50) * float64(weight)
		totalWeight += float64(weight)
	}
	if meal.Main != nil {
		add(*meal.Main, environment.MainWeight)
	}
	for _, dish := range meal.Starters {
		add(dish, environment.StarterWeight)
	}
	for _, dish := range meal.Desserts {
		add(dish, environment.DessertWeight)
	}
	if totalWeight == 0 {
		return 0
	}
	return float32(weightedChange / totalWeight)
}

// scoreMeal 在主菜交付时结算一次，配菜缺货也纳入整餐的 ±50 分上限。
// 统一结算避免分次加减、分别截断造成顺序偏差；排队扣分不属于餐品影响。
func (e *engine) scoreMeal(student *studentState) {
	change := mealSatisfactionChange(student.result.Meal, student.result.Student.Preference)
	change -= float32(student.missingSides) * environment.MissingSidePenalty
	change = min(environment.MaxMealSatisfactionChange, max(-environment.MaxMealSatisfactionChange, change))
	e.adjustSatisfaction(student, change)
}

func (e *engine) countWaiting(step time.Duration) {
	// 调用前已推进队列并开始可用服务；区域间移动不占模拟时间。
	// 前端圆点的走路动画不参与等待计时，只有因排队受阻才扣分。
	for _, student := range e.students {
		if student.result.Status != agents.Active {
			continue
		}
		area := student.result.Area
		if !area.IsWaitingArea() && student.started && !student.ready {
			continue // 实际取餐或接受服务的时间不算排队等待。
		}
		student.result.Waiting[area] += step
		student.result.TotalWait += step
		loss := environment.IndoorLossPerSecond
		if area == agents.Outside {
			loss = environment.OutdoorLossPerSecond
		}
		e.adjustSatisfaction(student, -loss*float32(step.Seconds()))
	}
}

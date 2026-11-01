package simulation

import "math/rand"

// weightedChoices 按权重随机排列候选菜品，不重复抽取同一菜品。
// 每轮 P(菜品) = (1 + 学生对该菜的个人评分) / 剩余候选的权重总和。
// 基础权重 1 让评分为零的菜也有机会被选中；不使用可能为负的满意度变化量。
// 库存不足时按此顺序尝试后续候选，仍然只领取允许的份数。
func weightedChoices(rng *rand.Rand, dishes []Dish, preference Preference) []string {
	remaining := append([]Dish(nil), dishes...)
	var choices []string
	for len(remaining) > 0 {
		var total float64
		for _, dish := range remaining {
			total += 1 + float64(dishScoreForStudent(dish, preference))
		}
		draw := rng.Float64() * total
		selected := len(remaining) - 1
		for i, dish := range remaining {
			draw -= 1 + float64(dishScoreForStudent(dish, preference))
			if draw < 0 {
				selected = i
				break
			}
		}
		choices = append(choices, remaining[selected].ID)
		remaining = append(remaining[:selected], remaining[selected+1:]...)
	}
	return choices
}

// chooseMeal 为每名学生生成一次候选顺序，避免每个时钟步都重新选菜。
// 素食学生排除含肉菜品；肉食学生先在肉类主菜中抽选，缺货后才尝试素食。
func chooseMeal(student Student, menu []Dish, seed int64) decision {
	studentSeed := int64(uint64(seed) ^ (uint64(student.ID) * 0x9e3779b97f4a7c15))
	rng := rand.New(rand.NewSource(studentSeed))
	var starters, mains, alternatives, desserts []Dish
	for _, dish := range menu {
		if student.Preference == Vegetarian && !dish.Vegetarian {
			continue
		}
		switch dish.Course {
		case Starter:
			starters = append(starters, dish)
		case Main:
			if student.Preference == MeatPreferred && dish.Vegetarian {
				alternatives = append(alternatives, dish)
			} else {
				mains = append(mains, dish)
			}
		case Dessert:
			desserts = append(desserts, dish)
		}
	}
	return decision{
		starterChoices: weightedChoices(rng, starters, student.Preference),
		mainChoices:    append(weightedChoices(rng, mains, student.Preference), weightedChoices(rng, alternatives, student.Preference)...),
		dessertChoices: weightedChoices(rng, desserts, student.Preference),
	}
}

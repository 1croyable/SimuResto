package environment

import agents "simuresto/SimuResto/Agents"

// DefaultMenu 返回独立菜单，每个菜品的字段逐行展开，便于阅读和调整。
// 参考页面中的 2026 年 10 月 19–23 日午餐及 Crous Market 餐饮供应。
// 此处合并多日菜品供模拟使用，不代表某一天全部同时供应。
// 菜品评分和份数均为模拟设定，不是 CROUS 官方评分或真实备餐量。
// 评分范围为 0–100，50 分中性；旧的加分值不直接作为百分制评分。
// Vegetarian 按模拟中的无肉、无鱼分类设置；具体原料配方未逐一核实。
// 学生总满意度始终限制在 0–100，每人最多领取一份主菜。
func DefaultMenu() []agents.Dish {
	menu := []agents.Dish{
		// 前菜。
		{
			ID:                "tabbouleh",
			Name:              "Taboulé", // 库斯库斯凉拌沙拉。
			Course:            agents.Starter,
			Vegetarian:        true,
			Portions:          StarterPortions / 4,
			SatisfactionScore: 65,
		},
		{
			ID:                "red_cabbage",
			Name:              "Chou rouge", // 红甘蓝沙拉。
			Course:            agents.Starter,
			Vegetarian:        true,
			Portions:          StarterPortions / 4,
			SatisfactionScore: 40,
		},
		{
			ID:                "chickpea_salad",
			Name:              "Salade de pois chiche", // 鹰嘴豆沙拉。
			Course:            agents.Starter,
			Vegetarian:        true,
			Portions:          StarterPortions / 4,
			SatisfactionScore: 65,
		},
		{
			ID:                "tuna_rice_salad",
			Name:              "Salade de riz au thon", // 金枪鱼米饭沙拉。
			Course:            agents.Starter,
			Vegetarian:        false,
			Portions:          StarterPortions / 4,
			SatisfactionScore: 75,
		},
	}
	menu = append(menu, DefaultMainDishes()...)
	menu = append(menu, []agents.Dish{
		// 甜点。
		{
			ID:                "seasonal_fruit",
			Name:              "Fruits de saison", // 时令水果。
			Course:            agents.Dessert,
			Vegetarian:        true,
			Portions:          DessertPortions / 4,
			SatisfactionScore: 50,
		},
		{
			ID:                "yogurt",
			Name:              "Yaourts variés", // 什锦口味酸奶。
			Course:            agents.Dessert,
			Vegetarian:        true,
			Portions:          DessertPortions / 4,
			SatisfactionScore: 55,
		},
		{
			ID:                "fruit_compote",
			Name:              "Compote", // 水果泥。
			Course:            agents.Dessert,
			Vegetarian:        true,
			Portions:          DessertPortions / 4,
			SatisfactionScore: 65,
		},
		{
			ID:                "brownie",
			Name:              "Brownie", // 巧克力布朗尼。
			Course:            agents.Dessert,
			Vegetarian:        true,
			Portions:          DessertPortions / 4,
			SatisfactionScore: 80,
		},
	}...)
	// 前菜和甜点各有四种，均分固定库存，余数分给排在前面的菜品。
	for i := 0; i < StarterPortions%4; i++ {
		menu[i].Portions++
	}
	for i := 0; i < DessertPortions%4; i++ {
		menu[len(menu)-4+i].Portions++
	}
	return menu
}

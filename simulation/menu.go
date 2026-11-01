package simulation

// DefaultMenu 返回独立菜单，每个菜品的字段逐行展开，便于阅读和调整。
// 参考页面中的 2026 年 10 月 19–23 日午餐及 Crous Market 餐饮供应。
// 此处合并多日菜品供模拟使用，不代表某一天全部同时供应。
// 菜品评分和份数均为模拟设定，不是 CROUS 官方评分或真实备餐量。
// 评分范围为 0–100，50 分中性；旧的加分值不直接作为百分制评分。
// Vegetarian 按模拟中的无肉、无鱼分类设置；具体原料配方未逐一核实。
// 学生总满意度始终限制在 0–100，每人最多领取一份主菜。
func DefaultMenu() []Dish {
	return []Dish{
		// 前菜。
		{
			ID:                "tabbouleh",
			Name:              "Taboulé", // 库斯库斯凉拌沙拉。
			Course:            Starter,
			Vegetarian:        true,
			Portions:          23,
			SatisfactionScore: 65,
		},
		{
			ID:                "red_cabbage",
			Name:              "Chou rouge", // 红甘蓝沙拉。
			Course:            Starter,
			Vegetarian:        true,
			Portions:          23,
			SatisfactionScore: 40,
		},
		{
			ID:                "chickpea_salad",
			Name:              "Salade de pois chiche", // 鹰嘴豆沙拉。
			Course:            Starter,
			Vegetarian:        true,
			Portions:          22,
			SatisfactionScore: 65,
		},
		{
			ID:                "tuna_rice_salad",
			Name:              "Salade de riz au thon", // 金枪鱼米饭沙拉。
			Course:            Starter,
			Vegetarian:        false,
			Portions:          22,
			SatisfactionScore: 75,
		},

		// 素食主菜。
		{
			ID:                "chili_sin_carne",
			Name:              "Chili sin carne", // 无肉辣豆炖菜。
			Course:            Main,
			Vegetarian:        true,
			Portions:          14,
			SatisfactionScore: 85,
		},
		{
			ID:                "omelette",
			Name:              "Omelette", // 法式煎蛋。
			Course:            Main,
			Vegetarian:        true,
			Portions:          13,
			SatisfactionScore: 70,
		},
		{
			ID:                "zucchini_crumble",
			Name:              "Crumble courgette et parmesan", // 西葫芦帕尔马干酪酥烤。
			Course:            Main,
			Vegetarian:        true,
			Portions:          13,
			SatisfactionScore: 85,
		},

		// 荤食主菜。
		{
			ID:                "tandoori_turkey",
			Name:              "Sauté de dinde tandoori", // 坦都里风味炒火鸡肉。
			Course:            Main,
			Vegetarian:        false,
			Portions:          20,
			SatisfactionScore: 90,
		},
		{
			ID:                "chicken_leg",
			Name:              "Cuisse de poulet", // 鸡腿。
			Course:            Main,
			Vegetarian:        false,
			Portions:          20,
			SatisfactionScore: 90,
		},
		{
			ID:                "breaded_fish",
			Name:              "Poisson pané au fromage", // 奶酪裹粉鱼排。
			Course:            Main,
			Vegetarian:        false,
			Portions:          20,
			SatisfactionScore: 85,
		},

		// 甜点。
		{
			ID:                "seasonal_fruit",
			Name:              "Fruits de saison", // 时令水果。
			Course:            Dessert,
			Vegetarian:        true,
			Portions:          23,
			SatisfactionScore: 50,
		},
		{
			ID:                "yogurt",
			Name:              "Yaourts variés", // 什锦口味酸奶。
			Course:            Dessert,
			Vegetarian:        true,
			Portions:          23,
			SatisfactionScore: 55,
		},
		{
			ID:                "fruit_compote",
			Name:              "Compote", // 水果泥。
			Course:            Dessert,
			Vegetarian:        true,
			Portions:          22,
			SatisfactionScore: 65,
		},
		{
			ID:                "brownie",
			Name:              "Brownie", // 巧克力布朗尼。
			Course:            Dessert,
			Vegetarian:        true,
			Portions:          22,
			SatisfactionScore: 80,
		},
	}
}

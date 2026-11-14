package environment

import (
	"fmt"
	"math"
	"strings"

	agents "simuresto/SimuResto/Agents"
)

// MainIngredients 是备餐界面的食材选择，评分为固定的模拟设定，界面只选择食材和份数。
type MainIngredients struct {
	Staples    []agents.MainIngredient
	Vegetables []agents.MainIngredient
	Proteins   []agents.MainIngredient
}

func DefaultMainIngredients() MainIngredients {
	return MainIngredients{
		Staples: []agents.MainIngredient{
			{Name: "米饭", Score: 65},
			{Name: "意面", Score: 75},
			{Name: "土豆泥", Score: 70},
			{Name: "Couscous", Score: 20},
		},
		Vegetables: []agents.MainIngredient{
			{Name: "普罗旺斯炖菜", Score: 75},
			{Name: "清蒸西兰花", Score: 50},
			{Name: "炒四季豆", Score: 60},
			{Name: "炖胡萝卜", Score: 45},
		},
		Proteins: []agents.MainIngredient{
			{Name: "烤鸡腿", Score: 90, ContainsMeat: true},
			{Name: "鱼排", Score: 80, ContainsMeat: true},
			{Name: "牛肉汉堡饼", Score: 85, ContainsMeat: true},
			{Name: "植物汉堡饼", Score: 70},
			{Name: "植物鸡块", Score: 75},
			{Name: "煎豆腐", Score: 50},
		},
	}
}

func DefaultMainDishes() []agents.Dish {
	food := DefaultMainIngredients()
	return []agents.Dish{
		agents.NewComposedMain("rice_plant_burger", agents.MainComponents{
			Staple: food.Staples[0], Vegetable: food.Vegetables[0], Protein: food.Proteins[3],
		}, 14),
		agents.NewComposedMain("pasta_plant_nuggets", agents.MainComponents{
			Staple: food.Staples[1], Vegetable: food.Vegetables[2], Protein: food.Proteins[4],
		}, 13),
		agents.NewComposedMain("couscous_tofu", agents.MainComponents{
			Staple: food.Staples[3], Vegetable: food.Vegetables[3], Protein: food.Proteins[5],
		}, 13),
		agents.NewComposedMain("rice_chicken", agents.MainComponents{
			Staple: food.Staples[0], Vegetable: food.Vegetables[0], Protein: food.Proteins[0],
		}, 20),
		agents.NewComposedMain("mash_fish", agents.MainComponents{
			Staple: food.Staples[2], Vegetable: food.Vegetables[1], Protein: food.Proteins[1],
		}, 20),
		agents.NewComposedMain("pasta_beef", agents.MainComponents{
			Staple: food.Staples[1], Vegetable: food.Vegetables[2], Protein: food.Proteins[2],
		}, 20),
	}
}

func validateMainComponents(dish agents.Dish) error {
	c := dish.Components
	if !c.IsSet() {
		return nil
	}
	if dish.Course != agents.Main {
		return fmt.Errorf("仅主菜可设置拼装食材：%s", dish.ID)
	}
	for _, part := range []agents.MainIngredient{c.Staple, c.Vegetable, c.Protein} {
		if strings.TrimSpace(part.Name) == "" || math.IsNaN(float64(part.Score)) || math.IsInf(float64(part.Score), 0) || part.Score < 0 || part.Score > 100 {
			return fmt.Errorf("主菜 %s 的三部分都需要名称和 0–100 分的评分", dish.ID)
		}
	}
	if c.Staple.ContainsMeat || c.Vegetable.ContainsMeat {
		return fmt.Errorf("主菜 %s 的主食和素菜部分须为素食", dish.ID)
	}
	return nil
}

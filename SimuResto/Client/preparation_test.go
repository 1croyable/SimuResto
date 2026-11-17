package client

import (
	"math"
	"testing"

	agents "simuresto/SimuResto/Agents"
)

func TestPreparationPreservesIndividualPortionsAndScores(t *testing.T) {
	g := settingsGame()
	g.addPreparation()
	index := len(g.preparation) - 1
	g.selectIngredient(index, 0, 3) // Couscous，固定 20 分。
	g.selectIngredient(index, 1, 3) // 炖胡萝卜，固定 45 分。
	g.selectIngredient(index, 2, 5) // 煎豆腐，固定 50 分，属于素食。
	g.preparation[index].portions = "7"
	g.preparation[0].portions = "3"
	want, err := g.preparedMainDishes()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.startConfigured(); err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	for _, dish := range want {
		found := false
		for _, stock := range g.snapshot.Stock {
			if stock.Dish.ID == dish.ID {
				found = true
				if stock.Remaining != dish.Portions || stock.Dish != dish {
					t.Fatalf("备餐组合被改写：%+v", stock)
				}
			}
		}
		if !found {
			t.Fatal("组合没有进入库存")
		}
	}
	if math.Abs(float64(want[index].SatisfactionScore)-235.0/6) > 0.0001 || !want[index].Vegetarian {
		t.Fatal("新组合应按固定食材评分得到约 39.17 分，且属于素食")
	}
	g.openSettings()
	reloaded, err := g.preparedMainDishes()
	if err != nil || len(reloaded) != len(want) {
		t.Fatal("重开设置丢失备餐")
	}
	for i := range want {
		if reloaded[i] != want[i] {
			t.Fatal("重开设置改变配方或数量")
		}
	}
}

func TestPreparationRejectsInvalidInputBeforeStarting(t *testing.T) {
	for _, value := range []string{"NaN", "10001", "-1", "", "1.5"} {
		g := settingsGame()
		g.preparation[0].portions = value
		if g.startConfigured() == nil || g.simulationGame != nil || g.settingsTab != preparationTab {
			t.Fatal("错误备餐不能启动模拟")
		}
	}
}

func TestPreparationSupportsEmptyMenuAndUniqueIDs(t *testing.T) {
	g := settingsGame()
	g.preparation = nil
	g.addPreparation()
	id := g.preparation[0].dish.ID
	g.addPreparation()
	if id == g.preparation[1].dish.ID {
		t.Fatal("组合 ID 重复")
	}
	g.preparation = nil
	if err := g.startConfigured(); err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	for _, stock := range g.snapshot.Stock {
		if stock.Dish.Course == agents.Main && (!stock.Dish.Vegetarian || stock.Dish.Portions != 0) {
			t.Fatal("空备餐应启用素食兜底")
		}
	}
}

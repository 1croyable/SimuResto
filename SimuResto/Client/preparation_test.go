package client

import (
	agents "simuresto/SimuResto/Agents"
	"testing"
)

func TestPreparationPreservesIndividualPortionsAndScores(t *testing.T) {
	g := settingsGame()
	g.addPreparation()
	index := len(g.preparation) - 1
	for g.preparation[index].dish.Components.Protein.ContainsMeat {
		g.cycleIngredient(index, 2)
	}
	g.preparation[index].fields = [4]string{"60", "30", "90", "7"}
	g.preparation[0].fields[3] = "3"
	g.syncPreparationTotals()
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
	if want[index].SatisfactionScore != 70 || !want[index].Vegetarian {
		t.Fatal("新组合评分或植物肉分类错误")
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
	for _, tc := range []struct {
		column int
		value  string
	}{
		{0, "NaN"}, {1, "101"}, {2, "-1"}, {2, ""}, {3, "1.5"}, {3, "-1"},
	} {
		g := settingsGame()
		g.preparation[0].fields[tc.column] = tc.value
		if g.startConfigured() == nil || g.simulationGame != nil {
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
	g.syncPreparationTotals()
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

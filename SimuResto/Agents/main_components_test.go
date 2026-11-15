package agents

import (
	"math"
	"testing"
)

func TestMainComponentsScoreAndClassification(t *testing.T) {
	for _, tc := range []struct {
		staple, vegetable, protein, want float32
	}{
		{50, 50, 50, 50}, {0, 0, 0, 0}, {100, 100, 100, 100},
		{60, 30, 90, 70}, {0, 0, 100, 50},
	} {
		components := MainComponents{
			Staple:    MainIngredient{Name: "米饭", Score: tc.staple},
			Vegetable: MainIngredient{Name: "炖菜", Score: tc.vegetable},
			Protein:   MainIngredient{Name: "植物肉", Score: tc.protein},
		}
		dish := NewComposedMain("test", components, 7)
		if math.Abs(float64(dish.SatisfactionScore-tc.want)) > 0.0001 || !dish.Vegetarian || dish.Portions != 7 {
			t.Fatalf("拼装评分、素食分类或数量错误：%+v", dish)
		}
		dish.Components.Protein.ContainsMeat = true
		if dish.Prepared().Vegetarian {
			t.Fatal("含肉配方应归类为荤食")
		}
	}
}

package engine

import (
	"context"
	"math"
	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
	"testing"
	"time"
)

func TestComposedMainNormalizesAndFeedsSatisfaction(t *testing.T) {
	c := environment.DefaultConfig()
	dish := agents.NewComposedMain("plant", agents.MainComponents{
		Staple:    agents.MainIngredient{Name: "米饭", Score: 60},
		Vegetable: agents.MainIngredient{Name: "炖菜", Score: 30},
		Protein:   agents.MainIngredient{Name: "植物汉堡肉", Score: 90},
	}, 1)
	// 模拟调用方只更新配方，残留旧评分和分类的情况。
	dish.SatisfactionScore, dish.Vegetarian = 1, false
	c.Menu = []agents.Dish{dish}
	report, err := Run(context.Background(), c, []agents.Student{
		{ID: 1, Preference: agents.Vegetarian, Satisfaction: 50, Patience: time.Hour},
		{ID: 2, Preference: agents.Vegetarian, Satisfaction: 50, Patience: time.Hour, ArrivalTime: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, student := range report.Students {
		if student.Status != agents.Completed || student.Satisfaction != 70 || student.Meal.Main.SatisfactionScore != 70 {
			t.Fatalf("组合评分未用于实际领取和结算：%+v", student)
		}
	}
	if report.Stock[0].Taken != 2 || report.Stock[0].Replenished != 1 {
		t.Fatal("拼装素食的兜底补货错误")
	}
	if c.Menu[0] != dish {
		t.Fatal("模拟修改了调用方配方")
	}
	for _, score := range []float32{-1, 101, float32(math.NaN()), float32(math.Inf(1))} {
		c.Menu[0].Components.Protein.Score = score
		if c.Validate(nil) == nil {
			t.Fatal("非法食材分数应被拒绝")
		}
	}
}

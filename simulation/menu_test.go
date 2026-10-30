package simulation

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestWeightedMenuChoices(t *testing.T) {
	menu := []Dish{
		{ID: "low", SatisfactionGain: 1},
		{ID: "high", SatisfactionGain: 7},
	}
	rng := rand.New(rand.NewSource(42))
	const draws = 10000
	high := 0
	for range draws {
		choices := weightedChoices(rng, menu)
		if len(choices) != 2 || choices[0] == choices[1] {
			t.Fatal("候选列表必须包含两种不同的菜品")
		}
		if choices[0] == "high" {
			high++
		}
	}
	// 理论概率为 8 / (2 + 8) = 80%，验证抽样结果而非要求每次都选高收益菜。
	if ratio := float64(high) / draws; math.Abs(ratio-0.8) > 0.02 {
		t.Fatalf("选菜比例不符合权重：%f", ratio)
	}
}

func TestMenuChoicesRespectDietAndSeed(t *testing.T) {
	menu := DefaultMenu()
	vegetarian := Student{ID: 1, Preference: Vegetarian}
	first := chooseMeal(vegetarian, menu, 42)
	if !reflect.DeepEqual(first, chooseMeal(vegetarian, menu, 42)) {
		t.Fatal("相同种子与学生必须得到相同候选顺序")
	}
	byID := make(map[string]Dish)
	for _, dish := range menu {
		byID[dish.ID] = dish
	}
	for _, choices := range [][]string{first.starterChoices, first.mainChoices, first.dessertChoices} {
		for _, id := range choices {
			if !byID[id].Vegetarian {
				t.Fatal("素食学生不能选择含肉菜品")
			}
		}
	}
	meat := chooseMeal(Student{ID: 2, Preference: MeatPreferred}, menu, 42)
	seenVegetarian := false
	for _, id := range meat.mainChoices {
		if byID[id].Vegetarian {
			seenVegetarian = true
		} else if seenVegetarian {
			t.Fatal("肉类主菜应在素食替代项之前")
		}
	}
	changed := false
	for seed := int64(43); seed < 60; seed++ {
		if !reflect.DeepEqual(first, chooseMeal(vegetarian, menu, seed)) {
			changed = true
		}
	}
	if !changed {
		t.Fatal("随机种子应影响选菜")
	}
}

func TestDishSatisfactionGainAndBounds(t *testing.T) {
	for _, initial := range []float32{20, 96} {
		config := DefaultConfig()
		config.Menu = []Dish{
			{ID: "starter", Course: Starter, Vegetarian: true, Portions: 1, SatisfactionGain: 2},
			{ID: "main", Course: Main, Vegetarian: true, Portions: 1, SatisfactionGain: 7},
			{ID: "dessert", Course: Dessert, Vegetarian: true, Portions: 1, SatisfactionGain: 4},
		}
		student := Student{ID: 1, Preference: Vegetarian, Satisfaction: initial, Patience: time.Minute, StartersWanted: 1, DessertsWanted: 1}
		report, err := Run(context.Background(), config, []Student{student})
		if err != nil {
			t.Fatal(err)
		}
		want := min(float32(100), initial+2+7+4+3*config.CourseVarietyBonus+config.PreferredMainBonus)
		if report.Students[0].Satisfaction != want {
			t.Fatalf("满意度为 %f，预期 %f", report.Students[0].Satisfaction, want)
		}
		if report.Stock[1].Taken != 1 || report.Students[0].Meal.Main == nil {
			t.Fatal("主菜必须恰好交付一份")
		}
	}
	for _, invalid := range []float32{-1, 101, float32(math.NaN()), float32(math.Inf(1))} {
		config := DefaultConfig()
		config.Menu[0].SatisfactionGain = invalid
		if _, err := Run(context.Background(), config, nil); err == nil {
			t.Fatal("非法菜品满意度收益应被拒绝")
		}
	}
}

func TestRandomMenuStockFallbackAndRepeatability(t *testing.T) {
	config := DefaultConfig()
	config.Menu = []Dish{
		{ID: "popular", Course: Main, Vegetarian: true, Portions: 0, SatisfactionGain: 100},
		{ID: "available", Course: Main, Vegetarian: true, Portions: 3, SatisfactionGain: 1},
	}
	students := make([]Student, 12)
	for i := range students {
		students[i] = Student{ID: i + 1, Preference: Vegetarian, Satisfaction: 50, Patience: time.Hour}
	}
	first, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("同一种子的模拟结果必须可复现")
	}
	served := make(map[int]int)
	for _, event := range first.Events {
		if event.Signal == MainServed {
			served[event.StudentID]++
		}
	}
	if len(served) != 3 || first.Stock[1].Taken != 3 || first.Stock[0].Taken != 0 {
		t.Fatal("缺货替代或库存扣减错误")
	}
	for _, count := range served {
		if count != 1 {
			t.Fatal("每名学生最多一份主菜")
		}
	}
	for _, result := range first.Students {
		if result.Satisfaction < 0 || result.Satisfaction > 100 {
			t.Fatal("满意度超出 0–100")
		}
	}
}

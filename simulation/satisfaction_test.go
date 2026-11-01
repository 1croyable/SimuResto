package simulation

import (
	"context"
	"math"
	"testing"
	"time"
)

func neutralMealConfig() Config {
	c := DefaultConfig()
	c.SelfServiceTime, c.MainServiceTime, c.PaymentTime = time.Second, time.Second, time.Second
	c.Menu = []Dish{
		{ID: "main", Course: Main, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
		{ID: "starter", Course: Starter, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
		{ID: "dessert", Course: Dessert, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
	}
	return c
}

func mealStudent() Student {
	return Student{ID: 1, Preference: Vegetarian, Patience: time.Hour, Satisfaction: 50, StartersWanted: 1, DessertsWanted: 1}
}

func runMeal(t *testing.T, c Config, student Student) StudentResult {
	t.Helper()
	report, err := Run(context.Background(), c, []Student{student})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.Completed != 1 || report.Students[0].Meal.Main == nil {
		t.Fatal("学生应领取一份主菜并完成付款")
	}
	for _, stock := range report.Stock {
		if stock.Remaining < 0 || stock.Reserved != 0 || stock.Remaining+stock.Taken != stock.Dish.Portions+stock.Replenished {
			t.Fatalf("库存不守恒：%+v", stock)
		}
	}
	return report.Students[0]
}

func TestMealSatisfactionUsesCenteredWeightedScores(t *testing.T) {
	standard := CourseWeights{Main: 2, Starter: 1, Dessert: 1}
	for _, tc := range []struct {
		name          string
		scores        [3]float32 // 主菜、前菜、甜点。
		weights       CourseWeights
		starters      int
		desserts      int
		initial, want float32
	}{
		{"中性", [3]float32{50, 50, 50}, standard, 1, 1, 50, 50},
		{"最低分", [3]float32{0, 0, 0}, standard, 1, 1, 50, 0},
		{"最高分", [3]float32{100, 100, 100}, standard, 1, 1, 50, 100},
		{"混合加分", [3]float32{80, 50, 20}, standard, 1, 1, 50, 57.5},
		{"混合扣分", [3]float32{20, 50, 80}, standard, 1, 1, 50, 42.5},
		{"调整权重", [3]float32{80, 50, 20}, CourseWeights{Main: 1, Starter: 1, Dessert: 4}, 1, 1, 50, 35},
		{"等比例权重", [3]float32{80, 50, 20}, CourseWeights{Main: 4, Starter: 2, Dessert: 2}, 1, 1, 50, 57.5},
		{"大权重不溢出", [3]float32{80, 50, 20}, CourseWeights{Main: 2e38, Starter: 1e38, Dessert: 1e38}, 1, 1, 50, 57.5},
		{"两份甜点", [3]float32{80, 100, 20}, standard, 0, 2, 50, 50},
		{"两份前菜", [3]float32{80, 20, 100}, standard, 2, 0, 50, 50},
		{"零权重忽略主菜", [3]float32{0, 70, 90}, CourseWeights{Starter: 1, Dessert: 1}, 1, 1, 50, 80},
		{"实际餐品权重全零", [3]float32{0, 0, 0}, CourseWeights{Dessert: 1}, 0, 0, 50, 50},
		{"只拿主菜", [3]float32{80, 0, 0}, standard, 0, 0, 50, 80},
		{"统一结算避免截断偏差", [3]float32{0, 100, 100}, standard, 1, 1, 95, 95},
		{"总满意度上限", [3]float32{100, 100, 100}, standard, 1, 1, 95, 100},
		{"总满意度下限", [3]float32{0, 0, 0}, standard, 1, 1, 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, student := neutralMealConfig(), mealStudent()
			c.SatisfactionWeights = tc.weights
			for i, score := range tc.scores {
				c.Menu[i].SatisfactionScore = score
			}
			student.StartersWanted, student.DessertsWanted = tc.starters, tc.desserts
			student.Satisfaction = tc.initial
			result := runMeal(t, c, student)
			if math.Abs(float64(result.Satisfaction-tc.want)) > 0.0001 {
				t.Fatalf("满意度 = %v，期望 %v", result.Satisfaction, tc.want)
			}
		})
	}
}

func TestMismatchedMainLoses25BeforeWeighting(t *testing.T) {
	student := mealStudent()
	student.Preference = MeatPreferred
	for _, tc := range []struct {
		score, want float32
	}{
		{80, 52.5}, {50, 37.5}, {25, 25}, {10, 25}, {0, 25},
	} {
		c := neutralMealConfig()
		c.Menu[0].SatisfactionScore = tc.score
		result := runMeal(t, c, student)
		if result.Satisfaction != tc.want {
			t.Fatalf("主菜评分 %v 减 25 后按 2:1:1 加权，实际 %v，期望 %v", tc.score, result.Satisfaction, tc.want)
		}
		if result.Meal.Main.SatisfactionScore != tc.score || c.Menu[0].SatisfactionScore != tc.score {
			t.Fatal("个人偏好不应修改公共菜单或餐盘记录的原始评分")
		}
	}
}

func TestFallbackMainParticipatesInMealScore(t *testing.T) {
	c := neutralMealConfig()
	c.Menu[0].Portions = 0
	c.Menu[1].SatisfactionScore, c.Menu[2].SatisfactionScore = 60, 80
	result := runMeal(t, c, mealStudent())
	if result.Meal.Main.ID != "main" || result.Satisfaction != 60 || result.TotalWait != 0 || result.ExitTime != 3*time.Second {
		t.Fatalf("兜底主菜也应服务一秒，再按整餐评分计分，不额外扣主菜缺货分：%+v", result)
	}
}

func TestFoodEffectIncludingShortagesIsCappedAt50(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		mainScore, missingPenalty, initial, want float32
	}{
		{"最低分加缺货", 0, 5, 80, 30},
		{"很大的缺货扣分", 50, 100, 80, 30},
		{"加减统一结算", 100, 5, 5, 45},
		{"最多加五十分", 100, 0, 20, 70},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, student := neutralMealConfig(), mealStudent()
			c.Menu[0].SatisfactionScore = tc.mainScore
			c.Menu[1].Portions, c.Menu[2].Portions = 0, 0
			c.MissingSidePenalty, student.Satisfaction = tc.missingPenalty, tc.initial
			result := runMeal(t, c, student)
			if result.Satisfaction != tc.want {
				t.Fatalf("餐品总影响不能超过 ±50，实际 %v，期望 %v", result.Satisfaction, tc.want)
			}
		})
	}
}

func TestFoodEffectIsSettledOnlyOnceAtMainDelivery(t *testing.T) {
	c := neutralMealConfig()
	c.Menu[1].Portions, c.Menu[2].Portions = 0, 0
	session, err := NewSession(c, []Student{mealStudent()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for step, want := range []float32{50, 40, 40} {
		if _, err := session.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got := session.Snapshot().Students[0].Satisfaction; got != want {
			t.Fatalf("第 %d 步满意度 %v，期望 %v；配菜缺货应随主菜交付统一结算一次", step+1, got, want)
		}
	}
}

func TestPersonalDishScoreOnlyAdjustsMismatchedMain(t *testing.T) {
	for _, tc := range []struct {
		preference Preference
		course     Course
		vegetarian bool
		want       float32
	}{
		{MeatPreferred, Main, true, 55},
		{MeatPreferred, Main, false, 80},
		{Vegetarian, Main, true, 80},
		{Vegetarian, Main, false, 55},
		{MeatPreferred, Starter, true, 80},
		{MeatPreferred, Dessert, true, 80},
	} {
		dish := Dish{Course: tc.course, Vegetarian: tc.vegetarian, SatisfactionScore: 80}
		if got := dishScoreForStudent(dish, tc.preference); got != tc.want {
			t.Fatalf("偏好 %v，菜品 %+v，评分 %v，期望 %v", tc.preference, dish, got, tc.want)
		}
	}
}

func TestSatisfactionConfigRejectsInvalidScoresAndWeights(t *testing.T) {
	for _, score := range []float32{-1, 101, float32(math.NaN()), float32(math.Inf(1))} {
		c := neutralMealConfig()
		c.Menu[0].SatisfactionScore = score
		if err := c.validate(nil); err == nil {
			t.Fatalf("应拒绝无效菜品评分 %v", score)
		}
	}
	for _, weights := range []CourseWeights{
		{}, {Main: -1}, {Starter: float32(math.NaN())}, {Dessert: float32(math.Inf(1))},
	} {
		c := neutralMealConfig()
		c.SatisfactionWeights = weights
		if err := c.validate(nil); err == nil {
			t.Fatalf("应拒绝无效权重 %+v", weights)
		}
	}
}

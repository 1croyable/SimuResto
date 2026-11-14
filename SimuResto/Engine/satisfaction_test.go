package engine

import (
	"context"
	"math"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func neutralMealConfig() environment.Config {
	c := environment.DefaultConfig()
	c.Menu = []agents.Dish{
		{ID: "main", Course: agents.Main, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
		{ID: "starter", Course: agents.Starter, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
		{ID: "dessert", Course: agents.Dessert, Vegetarian: true, Portions: 100, SatisfactionScore: 50},
	}
	return c
}

func mealStudent() agents.Student {
	return agents.Student{ID: 1, Preference: agents.Vegetarian, Temperature: agents.DefaultTemperature, Patience: time.Hour, Satisfaction: 50, StartersWanted: 1, DessertsWanted: 1}
}

func runMeal(t *testing.T, c environment.Config, student agents.Student) StudentResult {
	t.Helper()
	report, err := Run(context.Background(), c, []agents.Student{student})
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
	for _, tc := range []struct {
		name          string
		scores        [3]float32 // 主菜、前菜、甜点。
		starters      int
		desserts      int
		initial, want float32
	}{
		{"中性", [3]float32{50, 50, 50}, 1, 1, 50, 50},
		{"最低分", [3]float32{0, 0, 0}, 1, 1, 50, 0},
		{"最高分", [3]float32{100, 100, 100}, 1, 1, 50, 100},
		{"混合加分", [3]float32{80, 50, 20}, 1, 1, 50, 57.5},
		{"混合扣分", [3]float32{20, 50, 80}, 1, 1, 50, 42.5},
		{"两份甜点", [3]float32{80, 100, 20}, 0, 2, 50, 50},
		{"两份前菜", [3]float32{80, 20, 100}, 2, 0, 50, 50},
		{"只拿主菜", [3]float32{80, 0, 0}, 0, 0, 50, 80},
		{"统一结算避免截断偏差", [3]float32{0, 100, 100}, 1, 1, 95, 95},
		{"总满意度上限", [3]float32{100, 100, 100}, 1, 1, 95, 100},
		{"总满意度下限", [3]float32{0, 0, 0}, 1, 1, 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, student := neutralMealConfig(), mealStudent()
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
	student.Preference = agents.MeatPreferred
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
	if result.Meal.Main.ID != "main" || result.Satisfaction != 60 || result.TotalWait != 0 || result.ExitTime != environment.SelfServiceTime+environment.MainServiceTime+environment.PaymentTime {
		t.Fatalf("兜底主菜使用固定服务时长，不额外扣主菜缺货分，畅通时也不计行走等待：%+v", result)
	}
}

func TestFoodEffectIncludingShortagesIsCappedAt50(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mainScore, initial, want float32
		wantedSides              int
	}{
		{"最低分加缺货", 0, 80, 30, 2},
		{"缺两份配菜", 50, 80, 70, 2},
		{"加减统一结算", 100, 5, 45, 2},
		{"最多加五十分", 100, 20, 70, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, student := neutralMealConfig(), mealStudent()
			c.Menu[0].SatisfactionScore = tc.mainScore
			c.Menu[1].Portions, c.Menu[2].Portions = 0, 0
			student.Satisfaction = tc.initial
			student.StartersWanted, student.DessertsWanted = tc.wantedSides, 0
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
	game, err := NewGame(c, []agents.Student{mealStudent()})
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	steps := int((environment.SelfServiceTime + environment.MainServiceTime + environment.PaymentTime) / c.Tick)
	for step := 1; step <= steps; step++ {
		want := float32(50)
		if time.Duration(step)*c.Tick >= environment.SelfServiceTime+environment.MainServiceTime {
			want = 40
		}
		if _, err := game.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got := game.Snapshot().Students[0].Satisfaction; got != want {
			t.Fatalf("第 %d 步满意度 %v，期望 %v；配菜缺货应随主菜交付统一结算一次", step+1, got, want)
		}
	}
}

func TestPersonalDishScoreOnlyAdjustsMismatchedMain(t *testing.T) {
	for _, tc := range []struct {
		preference agents.Preference
		course     agents.Course
		vegetarian bool
		want       float32
	}{
		{agents.MeatPreferred, agents.Main, true, 55},
		{agents.MeatPreferred, agents.Main, false, 80},
		{agents.Vegetarian, agents.Main, true, 80},
		{agents.Vegetarian, agents.Main, false, 55},
		{agents.MeatPreferred, agents.Starter, true, 80},
		{agents.MeatPreferred, agents.Dessert, true, 80},
	} {
		dish := agents.Dish{Course: tc.course, Vegetarian: tc.vegetarian, SatisfactionScore: 80}
		if got := dishScoreForStudent(dish, tc.preference); got != tc.want {
			t.Fatalf("偏好 %v，菜品 %+v，评分 %v，期望 %v", tc.preference, dish, got, tc.want)
		}
	}
}

func TestSatisfactionConfigRejectsInvalidScores(t *testing.T) {
	for _, score := range []float32{-1, 101, float32(math.NaN()), float32(math.Inf(1))} {
		c := neutralMealConfig()
		c.Menu[0].SatisfactionScore = score
		if err := c.Validate(nil); err == nil {
			t.Fatalf("应拒绝无效菜品评分 %v", score)
		}
	}
}

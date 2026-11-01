package simulation

import (
	"context"
	"testing"
	"time"
)

func TestSoldOutMainsAlwaysServeLowestScoringVegetarianMain(t *testing.T) {
	c := neutralMealConfig()
	c.MainServiceTime = 3 * time.Second
	c.Menu = []Dish{
		{ID: "best", Course: Main, Vegetarian: true, SatisfactionScore: 90},
		{ID: "worst", Course: Main, Vegetarian: true, SatisfactionScore: 15},
		{ID: "middle", Course: Main, Vegetarian: true, SatisfactionScore: 40},
		{ID: "meat", Course: Main, Vegetarian: false, SatisfactionScore: 0},
		{ID: "starter", Course: Starter, Vegetarian: true, SatisfactionScore: 0},
	}
	students := make([]Student, 20)
	for i := range students {
		students[i] = Student{ID: i + 1, Preference: Vegetarian, Patience: time.Hour, Satisfaction: 80}
		if i%2 == 0 {
			students[i].Preference = MeatPreferred
		}
	}
	report, err := Run(context.Background(), c, students)
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.Completed != len(students) {
		t.Fatal("常规主菜为零仍必须让所有学生完成取餐")
	}
	for _, result := range report.Students {
		if result.Meal.Main == nil || result.Meal.Main.ID != "worst" || !result.Meal.Main.Vegetarian {
			t.Fatalf("必须领取最低评分的素食主菜：%+v", result.Meal.Main)
		}
	}
	orderedAt := map[int]time.Duration{}
	substituted, served := 0, 0
	for _, event := range report.Events {
		switch event.Signal {
		case MainSubstituted:
			substituted++
		case OrderPlaced:
			orderedAt[event.StudentID] = event.Time
		case MainServed:
			served++
			if event.Time-orderedAt[event.StudentID] != c.MainServiceTime {
				t.Fatal("兜底主菜也必须由工作人员按正常时长交付")
			}
		case FoodUnavailable:
			t.Fatal("不想拿配菜、主菜可兜底时，不应出现无餐可取事件")
		}
	}
	if substituted != 20 || served != 20 {
		t.Fatalf("应记录每次替换和交付，实际 %d / %d", substituted, served)
	}
	for _, stock := range report.Stock {
		if stock.Remaining+stock.Reserved+stock.Taken != stock.Dish.Portions+stock.Replenished || stock.Remaining < 0 {
			t.Fatalf("补货后库存仍须守恒：%+v", stock)
		}
		if stock.Dish.ID == "worst" && (stock.Replenished != 20 || stock.Taken != 20 || stock.Reserved != 0) {
			t.Fatalf("应自动补充并交付 20 份兜底主菜：%+v", stock)
		}
		if stock.Dish.ID != "worst" && stock.Replenished != 0 {
			t.Fatal("只允许兜底素食自动补货")
		}
	}
}

func TestRegularMainStockIsUsedBeforeFallback(t *testing.T) {
	stock := newInventory([]Dish{
		{ID: "preferred", Course: Main, Vegetarian: false, Portions: 1, SatisfactionScore: 80},
		{ID: "alternative", Course: Main, Vegetarian: true, Portions: 1, SatisfactionScore: 60},
		{ID: "fallback", Course: Main, Vegetarian: true, SatisfactionScore: 20},
	})
	choices := []string{"preferred", "alternative", "fallback"}
	for i, want := range []string{"preferred", "alternative", "fallback", "fallback"} {
		id, substituted := stock.reserveMain(choices)
		if id != want || substituted != (i >= 2) {
			t.Fatalf("第 %d 份应为 %s，实际 %s，兜底 %v", i+1, want, id, substituted)
		}
		stock.deliverMain(id)
	}
	if item := stock.items[stock.byID["fallback"]]; item.Replenished != 2 || item.Taken != 2 {
		t.Fatalf("常规库存用完后才补充两份兜底：%+v", item)
	}
}

func TestFallbackWorksWithoutVegetarianMenuAndAvoidsIDCollision(t *testing.T) {
	for _, menu := range [][]Dish{
		nil,
		{{ID: "meat", Course: Main, Portions: 10, SatisfactionScore: 90}},
		{{ID: "fallback_vegetarian", Course: Starter, Vegetarian: true, Portions: 5, SatisfactionScore: 90}},
	} {
		c, student := neutralMealConfig(), mealStudent()
		c.Menu = menu
		student.StartersWanted, student.DessertsWanted = 0, 0
		result := runMeal(t, c, student)
		if !result.Meal.Main.Vegetarian || result.Meal.Main.Course != Main || result.Meal.Main.SatisfactionScore != 0 {
			t.Fatalf("无素食主菜的菜单也必须提供 0 分备用素食：%+v", result.Meal.Main)
		}
		for _, dish := range menu {
			if result.Meal.Main.ID == dish.ID {
				t.Fatal("备用主菜不能复用其他菜品的 ID")
			}
		}
	}
}

func TestCloseReturnsFallbackReservation(t *testing.T) {
	c := neutralMealConfig()
	c.Menu[0].Portions = 0
	c.MainServiceTime = 8 * time.Second
	session, err := NewSession(c, []Student{mealStudent()})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for range 2 {
		if _, err := session.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	before := session.Snapshot().Stock[0]
	if before.Replenished != 1 || before.Reserved != 1 || before.Taken != 0 {
		t.Fatalf("关闭前应预留了一份补充的素食主菜：%+v", before)
	}
	session.Close()
	session.Close()
	after := session.Snapshot().Stock[0]
	if after.Replenished != 1 || after.Reserved != 0 || after.Taken != 0 || after.Remaining != 1 {
		t.Fatalf("取消后补充的主菜必须归还，不能凭空消失或重复归还：%+v", after)
	}
}

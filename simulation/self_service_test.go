package simulation

import (
	"reflect"
	"testing"
	"time"
)

func TestSidesShareAllowanceAndReplaceUnavailableCourses(t *testing.T) {
	for _, tc := range []struct {
		name             string
		wanted, stock    [2]int
		limits           [3]int // 前菜、甜点、两类合计。
		got              [2]int
		wantSatisfaction float32
	}{
		{"两份前菜", [2]int{2, 0}, [2]int{5, 5}, [3]int{2, 2, 2}, [2]int{2, 0}, 50},
		{"各一份", [2]int{1, 1}, [2]int{5, 5}, [3]int{2, 2, 2}, [2]int{1, 1}, 50},
		{"两份甜点", [2]int{0, 2}, [2]int{5, 5}, [3]int{2, 2, 2}, [2]int{0, 2}, 50},
		{"甜点替代前菜", [2]int{2, 0}, [2]int{0, 5}, [3]int{2, 2, 2}, [2]int{0, 2}, 50},
		{"前菜替代甜点", [2]int{0, 2}, [2]int{5, 0}, [3]int{2, 2, 2}, [2]int{2, 0}, 50},
		{"部分缺货", [2]int{2, 0}, [2]int{1, 5}, [3]int{2, 2, 2}, [2]int{1, 1}, 50},
		{"前菜禁止领取", [2]int{1, 1}, [2]int{5, 5}, [3]int{0, 2, 2}, [2]int{0, 2}, 50},
		{"甜点禁止领取", [2]int{1, 1}, [2]int{5, 5}, [3]int{2, 0, 2}, [2]int{2, 0}, 50},
		{"类别上限", [2]int{2, 0}, [2]int{5, 5}, [3]int{1, 1, 2}, [2]int{1, 1}, 50},
		{"合计上限", [2]int{2, 2}, [2]int{5, 5}, [3]int{2, 2, 2}, [2]int{2, 0}, 50},
		{"总上限为一", [2]int{1, 1}, [2]int{5, 5}, [3]int{2, 2, 1}, [2]int{1, 0}, 50},
		{"不拿配菜", [2]int{0, 0}, [2]int{5, 5}, [3]int{2, 2, 2}, [2]int{0, 0}, 50},
		{"不足两份", [2]int{1, 1}, [2]int{0, 1}, [3]int{2, 2, 2}, [2]int{0, 1}, 45},
		{"全部缺货", [2]int{1, 1}, [2]int{0, 0}, [3]int{2, 2, 2}, [2]int{0, 0}, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, student := neutralMealConfig(), mealStudent()
			student.StartersWanted, student.DessertsWanted = tc.wanted[0], tc.wanted[1]
			c.Menu[1].Portions, c.Menu[2].Portions = tc.stock[0], tc.stock[1]
			c.MaxStartersPerStudent, c.MaxDessertsPerStudent, c.MaxSelfServiceItems = tc.limits[0], tc.limits[1], tc.limits[2]
			result := runMeal(t, c, student)
			if got := [2]int{len(result.Meal.Starters), len(result.Meal.Desserts)}; got != tc.got {
				t.Fatalf("实际份数 %v，期望 %v", got, tc.got)
			}
			if result.Satisfaction != tc.wantSatisfaction {
				t.Fatalf("满意度 %v，期望 %v", result.Satisfaction, tc.wantSatisfaction)
			}
		})
	}
}

func TestSideSubstitutionRespectsVegetarianPreference(t *testing.T) {
	c := neutralMealConfig()
	c.Menu[1].Vegetarian, c.Menu[2].Portions = false, 0
	result := runMeal(t, c, mealStudent())
	if len(result.Meal.Starters)+len(result.Meal.Desserts) != 0 || result.Satisfaction != 40 {
		t.Fatal("素食学生不能用含肉前菜补足配菜")
	}
}

func TestGeneratedStudentsUseAllSideCombinationsReproducibly(t *testing.T) {
	students, err := GenerateStudents(100, 42, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, student := range students {
		if student.StartersWanted < 0 || student.DessertsWanted < 0 || student.StartersWanted+student.DessertsWanted != 2 {
			t.Fatal("默认配菜期望合计应为两份")
		}
		seen[student.StartersWanted] = true
	}
	again, _ := GenerateStudents(100, 42, time.Second)
	if len(seen) != 3 || !reflect.DeepEqual(students, again) {
		t.Fatal("应覆盖三种组合，相同种子应生成相同学生")
	}
}

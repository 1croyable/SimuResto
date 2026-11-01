package simulation

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestFreeRouteDoesNotChargeWalkingOrServiceTime(t *testing.T) {
	c := neutralMealConfig()
	c.SelfServiceTime, c.MainServiceTime, c.PaymentTime = 4*time.Second, 8*time.Second, 4*time.Second
	student := mealStudent()
	student.ArrivalTime = 10 * time.Second
	result := runMeal(t, c, student)
	if result.TotalWait != 0 || result.Satisfaction != 50 || result.ExitTime-result.ArrivalTime != 16*time.Second {
		t.Fatalf("畅通路线只经历服务时间，不应计入等待或扣分：%+v", result)
	}
	for _, area := range result.Path {
		if area == Outside || result.Waiting[area] != 0 {
			t.Fatal("室内有空位时不应在室外逗留或把移动计为等待")
		}
	}
}

func TestQueueLossOnlyChargesActualIndoorAndOutdoorWaiting(t *testing.T) {
	for _, insideCapacity := range []int{20, 1} {
		c := neutralMealConfig()
		c.Capacities.Inside, c.Capacities.SelfService = insideCapacity, 1
		c.SelfServiceTime = 4 * time.Second
		students := make([]Student, 6)
		for i := range students {
			students[i] = mealStudent()
			students[i].ID = i + 1
		}
		report, err := Run(context.Background(), c, students)
		if err != nil {
			t.Fatal(err)
		}
		if report.Students[1].Waiting[Inside] == 0 {
			t.Fatal("室内未满但自助区忙碌时，真实的室内排队仍应计时")
		}
		var outdoorWait time.Duration
		for _, result := range report.Students {
			outside := result.Waiting[Outside]
			outdoorWait += outside
			want := float32(50) - c.IndoorLossPerSecond*float32((result.TotalWait-outside).Seconds()) - c.OutdoorLossPerSecond*float32(outside.Seconds())
			if result.Status != Completed || math.Abs(float64(result.Satisfaction-want)) > 0.0001 {
				t.Fatalf("满意度应只扣除真实排队损失，实际 %v，期望 %v", result.Satisfaction, want)
			}
		}
		if (insideCapacity == 20 && outdoorWait != 0) || (insideCapacity == 1 && outdoorWait == 0) {
			t.Fatalf("室内容量 %d，实际室外等待 %v", insideCapacity, outdoorWait)
		}
	}
}

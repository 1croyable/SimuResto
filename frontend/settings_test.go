package frontend

import (
	"context"
	"errors"
	"testing"
	"time"

	"simuresto/simulation"
)

func settingsGame() *Game {
	c := simulation.DefaultConfig()
	c.Capacities.Payment, c.Cashiers = 2, 2
	c.Tick = 250 * time.Millisecond
	students, _ := simulation.GenerateStudents(100, c.Seed, time.Second)
	g := &Game{options: Options{Config: c, Students: students, ArrivalInterval: time.Second}, speed: 4, chinese: true, configuring: true}
	g.populateSettings()
	return g
}

func TestSettingsApplyBeforeStartingAgents(t *testing.T) {
	g := settingsGame()
	if g.session != nil {
		t.Fatal("设置页不能提前启动模拟")
	}
	g.settingValues[studentCount] = "12"
	g.settingValues[arrivalSeconds] = "0.5"
	g.settingValues[cashierCount] = "1"
	g.settingValues[serverCount] = "3"
	g.settingValues[mainSeconds] = "6.5"
	g.settingValues[starterStock] = "7"
	g.settingValues[vegetarianStock] = "5"
	g.settingValues[meatStock] = "9"
	g.settingValues[dessertStock] = "11"
	g.settingValues[playbackSpeed] = "2"
	if err := g.startConfigured(); err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.configuring || g.session == nil || len(g.options.Students) != 12 || g.speed != 2 {
		t.Fatal("开始按钮未应用设置")
	}
	if g.options.Config.MainServiceTime != 6500*time.Millisecond || g.options.Config.Cashiers != 1 || g.options.Config.Servers != 3 || g.options.Students[1].ArrivalTime != 500*time.Millisecond {
		t.Fatal("服务参数或到达间隔错误")
	}
	if g.snapshot.Duration != 0 {
		t.Fatal("开始前不应累计模拟时间")
	}
	counts := [4]int{}
	for _, stock := range g.snapshot.Stock {
		switch stock.Dish.Course {
		case simulation.Starter:
			counts[0] += stock.Remaining
		case simulation.Dessert:
			counts[3] += stock.Remaining
		case simulation.Main:
			if stock.Dish.Vegetarian {
				counts[1] += stock.Remaining
			} else {
				counts[2] += stock.Remaining
			}
		}
	}
	if counts != [4]int{7, 5, 9, 11} {
		t.Fatalf("实际库存没有采用设置：%v", counts)
	}
	g.step()
	old := g.session
	g.openSettings()
	if !g.configuring || g.settingValues[studentCount] != "12" {
		t.Fatal("返回设置应保留上一轮参数")
	}
	if done, err := old.Step(context.Background()); !done || !errors.Is(err, context.Canceled) {
		t.Fatal("返回设置必须结束上一轮智能体")
	}
}

func TestInvalidSettingsDoNotStartSimulation(t *testing.T) {
	for _, change := range []struct {
		key   settingKey
		value string
	}{
		{studentCount, ""}, {studentCount, "-1"}, {studentCount, "1.5"}, {cashierCount, "3"}, {tickSeconds, "0"}, {starterStock, "-1"}, {outdoorLoss, "0.01"}, {playbackSpeed, "NaN"},
	} {
		g := settingsGame()
		g.settingValues[change.key] = change.value
		if err := g.startConfigured(); err == nil {
			t.Fatalf("错误参数应被拒绝：%v", change)
		}
		if g.session != nil || !g.configuring {
			t.Fatal("错误参数不得启动后台模拟")
		}
	}
}

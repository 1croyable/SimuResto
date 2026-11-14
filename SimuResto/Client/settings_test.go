package client

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func settingsGame() *Game {
	c := environment.DefaultConfig()
	generator := agents.StudentGenerator{Seed: environment.RandomSeed, ArrivalInterval: environment.ArrivalInterval}
	students, _ := generator.Generate(environment.DefaultStudentCount)
	g := &Game{options: Options{Config: c, Students: students}, speed: environment.DefaultPlaybackSpeed, chinese: true, configuring: true}
	g.populateSettings()
	return g
}

func TestSettingsApplyBeforeStartingAgents(t *testing.T) {
	g := settingsGame()
	if g.simulationGame != nil {
		t.Fatal("设置页不能提前启动模拟")
	}
	g.settingValues[studentCount] = "12"
	g.settingValues[cashierCount] = "1"
	g.settingValues[tickSeconds] = "0.5"
	g.settingValues[maxMinutes] = "10"
	if err := g.startConfigured(); err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.configuring || g.simulationGame == nil || len(g.options.Students) != 12 ||
		g.options.Config.Cashiers != 1 || g.options.Config.Tick != 500*time.Millisecond || g.options.Config.MaxDuration != 10*time.Minute {
		t.Fatal("开始按钮未应用人数与时间设置")
	}
	if g.options.Students[1].ArrivalTime != environment.ArrivalInterval || g.speed != environment.DefaultPlaybackSpeed || g.snapshot.Duration != 0 {
		t.Fatal("到达间隔和默认速度应保持固定，开始前不应累计模拟时间")
	}
	for _, student := range g.snapshot.Students {
		if student.Student.Temperature != agents.DefaultTemperature {
			t.Fatal("开始模拟时应使用默认学生温度")
		}
	}
	counts := [2]int{}
	for _, stock := range g.snapshot.Stock {
		if stock.Dish.Course == agents.Starter {
			counts[0] += stock.Remaining
		}
		if stock.Dish.Course == agents.Dessert {
			counts[1] += stock.Remaining
		}
	}
	if counts != [2]int{environment.StarterPortions, environment.DessertPortions} {
		t.Fatalf("前菜、甜点应采用固定库存：%v", counts)
	}
	g.step()
	old := g.simulationGame
	g.openSettings()
	if !g.configuring || !reflect.DeepEqual(g.settingValues, []string{"12", "1", "0.5", "10"}) {
		t.Fatal("返回设置应保留上一轮人数与模拟时间")
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
		{studentCount, ""}, {studentCount, "-1"}, {studentCount, "1.5"}, {studentCount, "2001"},
		{cashierCount, "0"}, {cashierCount, "3"}, {cashierCount, "1.5"},
		{tickSeconds, "0"}, {tickSeconds, "NaN"}, {tickSeconds, "Inf"}, {tickSeconds, "3"},
		{maxMinutes, "0"}, {maxMinutes, "1.5"}, {maxMinutes, "121"},
	} {
		g := settingsGame()
		g.settingValues[change.key] = change.value
		if err := g.startConfigured(); err == nil {
			t.Fatalf("错误参数应被拒绝：%v", change)
		}
		if g.simulationGame != nil || !g.configuring {
			t.Fatal("错误参数不得启动后台模拟")
		}
	}
}

func TestRestoreDefaultsResetsSettingsAndPreparation(t *testing.T) {
	g := settingsGame()
	g.settingValues[studentCount] = "12"
	g.options.Config.Cashiers = 1
	g.speed = 16
	g.preparation = nil
	g.restoreDefaults()
	want := settingsGame()
	if !reflect.DeepEqual(g.options, want.options) || !reflect.DeepEqual(g.preparation, want.preparation) ||
		!reflect.DeepEqual(g.settingValues, want.settingValues) || g.speed != want.speed {
		t.Fatal("恢复默认应同时重置人数、时间、速度、窗口数量和备餐")
	}
}

func TestLiveCashierToggleKeepsCurrentSimulation(t *testing.T) {
	g := settingsGame()
	if err := g.startConfigured(); err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	for range 60 {
		g.step()
	}
	g.paused, g.speed = true, 8
	game, before := g.simulationGame, g.simulationGame.Report()
	if err := g.toggleCashiers(); err != nil {
		t.Fatal(err)
	}
	if game != g.simulationGame || !reflect.DeepEqual(before, g.simulationGame.Report()) ||
		!g.paused || g.speed != 8 || g.options.Config.Cashiers != 1 {
		t.Fatal("运行中切换收银不能重置模拟、改变暂停或播放速度")
	}
	g.Close()
	if err := g.toggleCashiers(); err == nil || g.options.Config.Cashiers != 1 {
		t.Fatal("结束后切换应被拒绝，不能改变前端窗口数量")
	}
}

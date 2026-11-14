package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

// 食堂的固定规则集中在这里，界面只调整人数、收银数量、逻辑步长和主菜备餐。
const (
	DefaultStudentCount  = 1200
	OpeningTime          = 11*time.Hour + 30*time.Minute
	ClosingTime          = 13*time.Hour + 45*time.Minute
	ArrivalPeakTime      = 12*time.Hour + 15*time.Minute
	ArrivalSigma         = 25 * time.Minute
	OpeningStudentRatio  = 0.02
	UniformArrivalRatio  = 0.20
	MinArrivalInterval   = time.Second
	DefaultTick          = 250 * time.Millisecond
	ServiceDuration      = ClosingTime - OpeningTime
	DefaultPlaybackSpeed = 4.0
	DefaultCashiers      = 2
	MaxCashiers          = 2

	OutdoorCapacity     = 80
	IndoorCapacity      = 25
	SelfServiceCapacity = 6
	MainServiceCapacity = 3
	Servers             = 2
	// 室外人数严格超过此容量百分比，才允许因耐心或满意度不足而离队。
	OutdoorAbandonmentThresholdPercent = 30

	// 每次服务开始时，在对应范围内均匀抽取本次耗时。
	MinSelfServiceTime = 4 * time.Second
	MaxSelfServiceTime = 10 * time.Second
	MinMainServiceTime = 6 * time.Second
	MaxMainServiceTime = 10 * time.Second
	MinPaymentTime     = 6 * time.Second
	MaxPaymentTime     = 12 * time.Second

	// CROUS 餐盘：一份主菜，前菜与甜点合计最多两份。
	MaxSideItems                      = 2
	MainWeight                float32 = 2
	StarterWeight             float32 = 1
	DessertWeight             float32 = 1
	IndoorLossPerSecond       float32 = 0.10
	OutdoorLossPerSecond      float32 = 0.35
	MissingSidePenalty        float32 = 5
	PreferenceMismatchPenalty float32 = 30
	MaxMealSatisfactionChange float32 = 50
	StarterPortions                   = 300
	DessertPortions                   = 350
)

// DefaultStudentGenerator 使用固定客流曲线，按本轮游戏的逻辑步长安排到达。
func DefaultStudentGenerator(tick time.Duration) agents.StudentGenerator {
	return agents.StudentGenerator{
		Tick:            tick,
		ArrivalDuration: ServiceDuration,
		ArrivalPeak:     ArrivalPeakTime - OpeningTime,
		ArrivalSigma:    ArrivalSigma,
		OpeningRatio:    OpeningStudentRatio,
		UniformRatio:    UniformArrivalRatio,
	}
}

// Config 只保留每轮可以改变的设置，收银窗口数可在运行中调整。
type Config struct {
	Tick     time.Duration
	Cashiers int
	Menu     []agents.Dish
}

func DefaultConfig() Config {
	return Config{
		Tick:     DefaultTick,
		Cashiers: DefaultCashiers,
		Menu:     DefaultMenu(),
	}
}

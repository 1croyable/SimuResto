package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

// 食堂的固定规则集中在这里，界面只调整人数、收银数量、模拟时间和主菜备餐。
const (
	DefaultStudentCount        = 300
	RandomSeed           int64 = 42
	ArrivalInterval            = time.Second
	DefaultTick                = 250 * time.Millisecond
	DefaultMaxDuration         = 15 * time.Minute
	DefaultPlaybackSpeed       = 4.0
	DefaultCashiers            = 2
	MaxCashiers                = 2

	OutdoorCapacity     = 50
	IndoorCapacity      = 20
	SelfServiceCapacity = 6
	MainServiceCapacity = 3
	Servers             = 2
	SelfServiceTime     = 4 * time.Second
	MainServiceTime     = 8 * time.Second
	PaymentTime         = 4 * time.Second

	// CROUS 餐盘：一份主菜，前菜与甜点合计最多两份。
	MaxSideItems                      = 2
	MainWeight                float32 = 2
	StarterWeight             float32 = 1
	DessertWeight             float32 = 1
	IndoorLossPerSecond       float32 = 0.08
	OutdoorLossPerSecond      float32 = 0.30
	MissingSidePenalty        float32 = 5
	PreferenceMismatchPenalty float32 = 25
	MaxMealSatisfactionChange float32 = 50
	StarterPortions                   = 90
	DessertPortions                   = 90
)

// Config 只保留每轮可以改变的设置，收银窗口数可在运行中调整。
type Config struct {
	Tick        time.Duration
	MaxDuration time.Duration
	Cashiers    int
	Menu        []agents.Dish
}

func DefaultConfig() Config {
	return Config{
		Tick:        DefaultTick,
		MaxDuration: DefaultMaxDuration,
		Cashiers:    DefaultCashiers,
		Menu:        DefaultMenu(),
	}
}

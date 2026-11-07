package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

type Capacities struct {
	Outside     int
	Inside      int
	SelfService int
	MainService int
	Payment     int
}

// CourseWeights 表示每一份菜在整餐评分中的相对权重。
type CourseWeights struct {
	Main    float32
	Starter float32
	Dessert float32
}

// Config 中的时长均为模拟时间，只有 TickInterval 表示真实播放间隔。
type Config struct {
	Capacities Capacities
	Seed       int64 // 选菜随机种子；每名学生使用独立随机源。

	// 时钟：Tick 为模拟步长，TickInterval 为真实播放间隔。
	Tick         time.Duration
	TickInterval time.Duration
	MaxDuration  time.Duration

	// 各环节服务时长与工作人员数量。
	SelfServiceTime time.Duration
	MainServiceTime time.Duration
	PaymentTime     time.Duration
	Servers         int
	Cashiers        int

	// 前菜和甜点共享总份数，可以只取其中一类；主菜最多一份。
	MaxStartersPerStudent int
	MaxDessertsPerStudent int
	MaxSelfServiceItems   int

	// 菜品按权重计算整餐评分，50 分为中性；配菜缺货计入整餐，等待另计。
	SatisfactionWeights  CourseWeights
	IndoorLossPerSecond  float32
	OutdoorLossPerSecond float32
	MissingSidePenalty   float32
	Menu                 []agents.Dish
}

func DefaultConfig() Config {
	return Config{
		Seed:                  42,
		Capacities:            Capacities{Outside: 50, Inside: 20, SelfService: 6, MainService: 3, Payment: 1},
		Tick:                  time.Second,
		MaxDuration:           15 * time.Minute,
		SelfServiceTime:       4 * time.Second,
		MainServiceTime:       8 * time.Second,
		PaymentTime:           4 * time.Second,
		Servers:               2,
		Cashiers:              1,
		MaxStartersPerStudent: 2,
		MaxDessertsPerStudent: 2,
		MaxSelfServiceItems:   2,
		SatisfactionWeights:   CourseWeights{Main: 2, Starter: 1, Dessert: 1},
		IndoorLossPerSecond:   0.08,
		OutdoorLossPerSecond:  0.30,
		MissingSidePenalty:    5,
		Menu:                  DefaultMenu(),
	}
}

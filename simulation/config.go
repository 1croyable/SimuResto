package simulation

import "time"

type Capacities struct {
	Outside     int
	Inside      int
	SelfService int
	MainService int
	Payment     int
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

	// 配菜领取上限；主菜固定为每名学生最多一份。
	MaxStartersPerStudent int
	MaxDessertsPerStudent int
	MaxSelfServiceItems   int

	// 满意度变化，单位为分数或每模拟秒扣除的分数。
	IndoorLossPerSecond    float32
	OutdoorLossPerSecond   float32
	PreferredMainBonus     float32
	AlternativeMainPenalty float32
	MissingMainPenalty     float32
	MissingSidePenalty     float32
	CourseVarietyBonus     float32
	Menu                   []Dish
}

func DefaultConfig() Config {
	return Config{
		Seed:                   42,
		Capacities:             Capacities{Outside: 50, Inside: 20, SelfService: 6, MainService: 3, Payment: 1},
		Tick:                   time.Second,
		MaxDuration:            15 * time.Minute,
		SelfServiceTime:        4 * time.Second,
		MainServiceTime:        8 * time.Second,
		PaymentTime:            4 * time.Second,
		Servers:                2,
		Cashiers:               1,
		MaxStartersPerStudent:  1,
		MaxDessertsPerStudent:  1,
		MaxSelfServiceItems:    2,
		IndoorLossPerSecond:    0.08,
		OutdoorLossPerSecond:   0.30,
		PreferredMainBonus:     3,
		AlternativeMainPenalty: 10,
		MissingMainPenalty:     25,
		MissingSidePenalty:     5,
		CourseVarietyBonus:     2,
		Menu:                   DefaultMenu(),
	}
}

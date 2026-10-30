// Package simulation 使用 goroutine 模拟无图形界面的 CROUS 食堂服务。
package simulation

import "time"

type Student struct {
	ID             int
	Preference     Preference
	ArrivalTime    time.Duration
	Patience       time.Duration //是否改成数值比较好？
	Satisfaction   float32       //满分100，0-100
	StartersWanted int
	DessertsWanted int
}

type Staff struct {
	ID          int
	Role        Role
	ServiceTime time.Duration
}

type Dish struct {
	ID               string
	Name             string
	Course           Course
	Vegetarian       bool
	Portions         int
	SatisfactionGain float32 // 每领取一份增加的满意度，也决定选菜权重。
}

// Stock 记录每日库存，满足 Remaining + Reserved + Taken == Dish.Portions。
// 主菜接单时预留，工作人员交付后才计入已领取份数。
type Stock struct {
	Dish      Dish
	Remaining int //还在柜台内的部分
	Reserved  int //正在打饭
	Taken     int //已经领取
}

type Meal struct {
	Starters []Dish
	Main     *Dish // 最多一份主菜
	Desserts []Dish
}

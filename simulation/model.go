// Package simulation 使用 goroutine 模拟无图形界面的 CROUS 食堂服务。
package simulation

import "time"

type Student struct {
	ID             int
	Preference     Preference
	ArrivalTime    time.Duration
	Patience       time.Duration //是否改成数值比较好？
	Satisfaction   float32       //满分100，0-100
	StartersWanted int           // 优先想拿的前菜份数，缺货或受限时允许换成甜点。
	DessertsWanted int           // 优先想拿的甜点份数，缺货或受限时允许换成前菜。
}

type Staff struct {
	ID          int
	Role        Role
	ServiceTime time.Duration
}

type Dish struct {
	ID                string
	Name              string
	Course            Course
	Vegetarian        bool
	Portions          int
	SatisfactionScore float32 // 菜品评分为 0–100，50 分中性；也影响选菜概率。
}

// Stock 满足 Remaining + Reserved + Taken == Dish.Portions + Replenished。
// 主菜接单时预留，工作人员交付后才计入已领取份数。
type Stock struct {
	Dish        Dish
	Remaining   int //还在柜台内的部分
	Reserved    int //正在打饭
	Taken       int //已经领取
	Replenished int // 兜底素食自动补充的份数，普通菜品为零。
}

type Meal struct {
	Starters []Dish
	Main     *Dish // 最多一份主菜
	Desserts []Dish
}

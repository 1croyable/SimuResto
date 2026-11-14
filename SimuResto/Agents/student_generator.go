package agents

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// StudentGenerator 保存学生生成规则，供命令行和图形前端共用。
// 只负责生成学生资料，不管理入场、队列或学生 goroutine。
type StudentGenerator struct {
	Tick            time.Duration // 与本轮游戏一致的逻辑步长。
	ArrivalDuration time.Duration // 从开门到停止接纳新学生的时长。
	ArrivalPeak     time.Duration // 客流峰值距开门的时间。
	ArrivalSigma    time.Duration // 到达时间的标准差。
	OpeningRatio    float64       // 开门时已经到达的学生比例，包含在总人数内。
	UniformRatio    float64       // 后续客流中均匀分布的占比，其余采用正态分布。
}

// Generate 提前创建学生资料，并按每个游戏刻度的累计客流安排到达。
// 多次调用用于创建不同轮次的模拟，不用于向同一轮模拟追加学生。
func (g StudentGenerator) Generate(count int) ([]Student, error) {
	if count < 0 || g.Tick <= 0 || g.ArrivalDuration <= 0 || g.ArrivalSigma <= 0 || g.ArrivalPeak < 0 || g.ArrivalPeak > g.ArrivalDuration {
		return nil, fmt.Errorf("学生人数不能为负，逻辑步长、营业时长和标准差必须为正，峰值必须位于营业时间内")
	}
	if !(g.OpeningRatio >= 0 && g.OpeningRatio <= 1) {
		return nil, fmt.Errorf("开门时的学生比例必须在 0 到 1 之间")
	}
	if !(g.UniformRatio >= 0 && g.UniformRatio <= 1) {
		return nil, fmt.Errorf("均匀客流比例必须在 0 到 1 之间")
	}
	students := make([]Student, count)
	for i := range students {
		preference := MeatPreferred
		if rand.Float64() < 0.35 {
			preference = Vegetarian
		}
		students[i] = Student{
			ID: i + 1, Preference: preference,
			Patience:     time.Duration(40+rand.IntN(121)) * time.Second,
			Satisfaction: float32(80 + rand.IntN(16)),
			Temperature:  DefaultTemperature,
		}
		// 两份前菜、各一份、两份甜点的概率分别为 20%、60%、20%。
		sideChoice := rand.Float64()
		switch {
		case sideChoice < 0.2:
			students[i].StartersWanted = 2
		case sideChoice < 0.8:
			students[i].StartersWanted = 1
		default:
			students[i].StartersWanted = 0
		}
		students[i].DessertsWanted = 2 - students[i].StartersWanted
	}
	g.scheduleArrivals(students)
	return students, nil
}

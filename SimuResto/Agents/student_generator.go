package agents

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// StudentGenerator 保存学生生成规则，供命令行和图形前端共用。
// 只负责生成学生资料，不管理入场、队列或学生 goroutine。
type StudentGenerator struct {
	ArrivalInterval time.Duration // 当前采用固定到达间隔，0 表示同时到达。
}

// Generate 每次随机生成一批学生，ID 从 1 开始，到达时间从 0 开始。
// 多次调用用于创建不同轮次的模拟，不用于向同一轮模拟追加学生。
func (g StudentGenerator) Generate(count int) ([]Student, error) {
	arrivalInterval := g.ArrivalInterval
	if count < 0 || arrivalInterval < 0 {
		return nil, fmt.Errorf("student count and arrival interval cannot be negative")
	}
	if count > 1 && int64(arrivalInterval) > math.MaxInt64/int64(count-1) {
		return nil, fmt.Errorf("arrival schedule overflows time.Duration")
	}
	students := make([]Student, count)
	for i := range students {
		preference := MeatPreferred
		if rand.Float64() < 0.35 {
			preference = Vegetarian
		}
		students[i] = Student{
			ID: i + 1, Preference: preference, ArrivalTime: time.Duration(i) * arrivalInterval,
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
	return students, nil
}

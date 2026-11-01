package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// StudentGenerator 保存学生生成规则，供命令行和图形前端共用。
// 只负责生成学生资料，不管理入场、队列或学生 goroutine。
type StudentGenerator struct {
	Seed            int64         // 随机种子，相同参数生成相同学生。
	ArrivalInterval time.Duration // 当前采用固定到达间隔，0 表示同时到达。
}

// Generate 每次使用种子重新生成一批学生，ID 从 1 开始，到达时间从 0 开始。
// 多次调用用于创建不同轮次的模拟，不用于向同一轮模拟追加学生。
func (g StudentGenerator) Generate(count int) ([]Student, error) {
	arrivalInterval := g.ArrivalInterval
	if count < 0 || arrivalInterval < 0 {
		return nil, fmt.Errorf("student count and arrival interval cannot be negative")
	}
	if count > 1 && int64(arrivalInterval) > math.MaxInt64/int64(count-1) {
		return nil, fmt.Errorf("arrival schedule overflows time.Duration")
	}
	rng := rand.New(rand.NewSource(g.Seed))
	students := make([]Student, count)
	for i := range students {
		preference := MeatPreferred
		if rng.Float64() < 0.35 {
			preference = Vegetarian
		}
		students[i] = Student{
			ID: i + 1, Preference: preference, ArrivalTime: time.Duration(i) * arrivalInterval,
			Patience:     time.Duration(40+rng.Intn(121)) * time.Second,
			Satisfaction: float32(80 + rng.Intn(16)),
			Temperature:  DefaultTemperature,
		}
		// 默认共想拿两份配菜，三种组合都可能出现。
		students[i].StartersWanted = rng.Intn(3)
		students[i].DessertsWanted = 2 - students[i].StartersWanted
	}
	return students, nil
}

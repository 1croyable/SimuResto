package agents

import (
	"math"
	"time"
)

// 混合均匀客流和营业时间内归一化的正态客流，计算每个刻度新增人数。
func (g StudentGenerator) scheduleArrivals(students []Student) {
	cdf := func(t time.Duration) float64 {
		z := float64(t-g.ArrivalPeak) / float64(g.ArrivalSigma)
		return (1 + math.Erf(z/math.Sqrt2)) / 2
	}
	start, end := cdf(0), cdf(g.ArrivalDuration)
	openingCount := 0
	if g.OpeningRatio > 0 && len(students) > 0 {
		// 开门即有少量学生；人数很少时至少安排一人，不额外增加总人数。
		openingCount = max(1, int(math.Round(float64(len(students))*g.OpeningRatio)))
	}
	// 开门时到达的资料保持 ArrivalTime 为零，其余学生按曲线陆续到达。
	assigned := openingCount
	remaining := len(students) - openingCount
	for t := time.Duration(0); t < g.ArrivalDuration; {
		t += min(g.Tick, g.ArrivalDuration-t)
		// 累计取整后求差，避免每个刻度分别取整导致少量客流被舍弃。
		normalFraction := (cdf(t) - start) / (end - start)
		uniformFraction := float64(t) / float64(g.ArrivalDuration)
		fraction := g.UniformRatio*uniformFraction + (1-g.UniformRatio)*normalFraction
		target := min(len(students), openingCount+int(math.Round(float64(remaining)*fraction)))
		for assigned < target {
			students[assigned].ArrivalTime = t
			assigned++
		}
	}
}

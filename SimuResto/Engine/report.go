package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

type StudentResult struct {
	Student      agents.Student
	Status       agents.Status
	Area         agents.Area
	ArrivalTime  time.Duration // 从隐藏缓冲区放行、实际抵达食堂的时间。
	ExitTime     time.Duration
	Satisfaction float32
	TotalWait    time.Duration
	Waiting      map[agents.Area]time.Duration
	Path         []agents.Area
	Meal         agents.Meal
}

type StaffResult struct {
	Staff       agents.Staff
	Served      int
	BusyTime    time.Duration
	Utilization float32
}

type Metrics struct {
	Completed           int
	Abandoned           int
	Cancelled           int
	AverageWait         time.Duration
	AverageWaitByArea   map[agents.Area]time.Duration
	AverageSatisfaction float64
	AbandonmentRate     float64
	MealsPerMinute      float64
}

type Report struct {
	Duration      time.Duration
	Students      []StudentResult
	Staff         []StaffResult
	Stock         []agents.Stock
	PeakOccupancy map[agents.Area]int
	Events        []Event
	Metrics       Metrics
}

func (e *engine) report() Report {
	report := Report{
		Duration: e.now, Stock: e.stock.items, Events: e.stats.events, PeakOccupancy: e.stats.peakOccupancy,
		Metrics: Metrics{AverageWaitByArea: make(map[agents.Area]time.Duration)},
	}
	var totalWait time.Duration
	var totalSatisfaction float64
	admitted := 0
	arrivedCount := 0
	visitors := make(map[agents.Area]int)
	for _, student := range e.students {
		result := student.result
		report.Students = append(report.Students, result)
		switch result.Status {
		case agents.Completed:
			report.Metrics.Completed++
		case agents.Abandoned:
			report.Metrics.Abandoned++
		case agents.Cancelled:
			report.Metrics.Cancelled++
		}
		// 已进入队列或到达后直接放弃的学生，都计入放弃率的分母。
		if len(result.Path) > 0 || result.Status == agents.Abandoned {
			arrivedCount++
		}
		if len(result.Path) > 0 {
			admitted++
			totalWait += result.TotalWait
			totalSatisfaction += float64(result.Satisfaction)
		}
		for _, area := range result.Path {
			visitors[area]++
			report.Metrics.AverageWaitByArea[area] += result.Waiting[area]
		}
	}
	for _, area := range agents.ServiceRoute() {
		if visitors[area] > 0 {
			report.Metrics.AverageWaitByArea[area] /= time.Duration(visitors[area])
		} else {
			report.Metrics.AverageWaitByArea[area] = 0
		}
	}
	if admitted > 0 {
		report.Metrics.AverageWait = totalWait / time.Duration(admitted)
		report.Metrics.AverageSatisfaction = totalSatisfaction / float64(admitted)
	}
	if arrivedCount > 0 {
		report.Metrics.AbandonmentRate = float64(report.Metrics.Abandoned) / float64(arrivedCount)
	}
	if e.now > 0 {
		report.Metrics.MealsPerMinute = float64(report.Metrics.Completed) / e.now.Minutes()
	}
	for _, worker := range e.staff {
		result := worker.result
		if e.now > 0 {
			result.Utilization = float32(float64(result.BusyTime) / float64(e.now))
		}
		report.Staff = append(report.Staff, result)
	}
	return report
}

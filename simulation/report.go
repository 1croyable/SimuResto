package simulation

import "time"

type StudentResult struct {
	Student      Student
	Status       Status
	Area         Area
	ArrivalTime  time.Duration
	ExitTime     time.Duration
	Satisfaction float32
	TotalWait    time.Duration
	Waiting      map[Area]time.Duration
	Path         []Area
	Meal         Meal
}

type StaffResult struct {
	Staff       Staff
	Served      int
	BusyTime    time.Duration
	Utilization float32
}

type Metrics struct {
	Completed           int
	Abandoned           int
	Rejected            int
	Cancelled           int
	AverageWait         time.Duration
	AverageWaitByArea   map[Area]time.Duration
	AverageSatisfaction float64
	AbandonmentRate     float64
	MealsPerMinute      float64
}

type Report struct {
	Duration      time.Duration
	Students      []StudentResult
	Staff         []StaffResult
	Stock         []Stock
	PeakOccupancy map[Area]int
	Events        []Event
	Metrics       Metrics
}

func (e *engine) report() Report {
	report := Report{
		Duration: e.now, Stock: e.stock.items, Events: e.events, PeakOccupancy: e.peaks,
		Metrics: Metrics{AverageWaitByArea: make(map[Area]time.Duration)},
	}
	var totalWait time.Duration
	var totalSatisfaction float64
	admitted := 0
	visitors := make(map[Area]int)
	for _, student := range e.students {
		result := student.result
		report.Students = append(report.Students, result)
		switch result.Status {
		case Completed:
			report.Metrics.Completed++
		case Abandoned:
			report.Metrics.Abandoned++
		case Rejected:
			report.Metrics.Rejected++
		case Cancelled:
			report.Metrics.Cancelled++
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
	for _, area := range ServiceRoute() {
		if visitors[area] > 0 {
			report.Metrics.AverageWaitByArea[area] /= time.Duration(visitors[area])
		} else {
			report.Metrics.AverageWaitByArea[area] = 0
		}
	}
	if admitted > 0 {
		report.Metrics.AverageWait = totalWait / time.Duration(admitted)
		report.Metrics.AverageSatisfaction = totalSatisfaction / float64(admitted)
		report.Metrics.AbandonmentRate = float64(report.Metrics.Abandoned) / float64(admitted)
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

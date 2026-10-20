package simulation

import "time"

func (e *engine) report() Report {
	report := Report{
		Duration: e.now, Stock: e.stock, Events: e.events, PeakOccupancy: e.peaks,
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
			totalSatisfaction += result.Satisfaction
		}
		for _, area := range result.Path {
			visitors[area]++
			report.Metrics.AverageWaitByArea[area] += result.Waiting[area]
		}
	}
	for _, area := range route {
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
			result.Utilization = float64(result.BusyTime) / float64(e.now)
		}
		report.Staff = append(report.Staff, result)
	}
	return report
}

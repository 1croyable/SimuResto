package simulation

import "time"

// 满意度使用 float32；时间使用 time.Duration。
func (e *engine) adjustSatisfaction(student *studentState, amount float32) {
	student.result.Satisfaction = min(100, max(0, student.result.Satisfaction+amount))
}

func (e *engine) countWaiting(step time.Duration) {
	for _, student := range e.students {
		if student.result.Status != Active {
			continue
		}
		area := student.result.Area
		if !area.IsWaitingArea() && student.started && !student.ready {
			continue // 实际取餐或接受服务的时间不算排队等待。
		}
		student.result.Waiting[area] += step
		student.result.TotalWait += step
		loss := e.config.IndoorLossPerSecond
		if area == Outside {
			loss = e.config.OutdoorLossPerSecond
		}
		e.adjustSatisfaction(student, -loss*float32(step.Seconds()))
	}
}

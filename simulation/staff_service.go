package simulation

import "time"

func (e *engine) assignStaff() {
	for _, worker := range e.staff {
		if worker.job != nil {
			continue
		}
		area := MainService
		if worker.result.Staff.Role == Cashier {
			area = Payment
		}
		for _, id := range e.queues[area].IDs() {
			student := e.byID[id]
			if student.started {
				continue
			}
			student.started = true
			task := &job{studentID: id, fresh: true}
			signal := PaymentStarted
			if area == MainService {
				task.dishID = e.stock.reserveMain(student.plan.mainChoices)
				if task.dishID == "" {
					student.ready = true
					e.adjustSatisfaction(student, -e.config.MissingMainPenalty)
					e.emit(FoodUnavailable, id, 0, area, area)
					continue
				}
				signal = OrderPlaced
			}
			worker.job = task
			var dishes []string
			if task.dishID != "" {
				dishes = []string{task.dishID}
			}
			e.emit(signal, id, worker.result.Staff.ID, area, area, dishes...)
			break
		}
	}
}

func (e *engine) advanceStaffService(step time.Duration) {
	// 先向所有工作人员发送时钟更新，再按工作人员顺序处理回复，
	// 保证事件记录不受 goroutine 调度顺序影响。
	for _, worker := range e.staff {
		if worker.job == nil {
			continue
		}
		update := workTick{elapsed: step}
		if worker.job.fresh {
			update.start = &assignment{studentID: worker.job.studentID, duration: worker.result.Staff.ServiceTime}
			worker.job.fresh = false
		}
		worker.agent.in <- update
	}
	for _, worker := range e.staff {
		if worker.job == nil {
			continue
		}
		result := <-worker.agent.out
		worker.result.BusyTime += result.busy
		if result.signal == Progress {
			continue
		}
		student := e.byID[result.studentID]
		student.ready = true
		worker.result.Served++
		var dishes []string
		if result.signal == MainServed {
			dish := e.stock.deliverMain(worker.job.dishID)
			student.result.Meal.Main = &dish
			dishes = []string{dish.ID}
			change := e.config.PreferredMainBonus
			if student.result.Student.Preference == MeatPreferred && dish.Vegetarian {
				change = -e.config.AlternativeMainPenalty
			}
			e.adjustSatisfaction(student, change+e.config.CourseVarietyBonus+dish.SatisfactionGain)
		}
		e.emit(result.signal, result.studentID, worker.result.Staff.ID, student.result.Area, student.result.Area, dishes...)
		worker.job = nil
	}
}

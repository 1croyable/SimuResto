package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func (e *engine) workerOpen(worker *staffState) bool {
	return worker.result.Staff.Role == agents.Server || worker.result.Staff.ID <= environment.Servers+e.config.Cashiers
}

func (e *engine) assignStaff() {
	for _, worker := range e.staff {
		if worker.job != nil || !e.workerOpen(worker) {
			continue
		}
		area := agents.MainService
		if worker.result.Staff.Role == agents.Cashier {
			area = agents.Payment
		}
		for _, id := range e.queues[area].IDs() {
			student := e.ID2Student[id]
			if student.started {
				continue
			}
			student.started = true
			task := &job{studentID: id, fresh: true}
			signal := agents.PaymentStarted
			if area == agents.MainService {
				var substituted bool
				task.dishID, substituted = e.reserveStudentMain(student)
				if substituted {
					e.emit(agents.MainSubstituted, id, worker.result.Staff.ID, area, area, task.dishID)
				}
				signal = agents.OrderPlaced
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

func (e *engine) stepStaffService(step time.Duration) {
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
		if result.signal == agents.Progress {
			continue
		}
		student := e.ID2Student[result.studentID]
		student.ready = true
		worker.result.Served++
		var dishes []string
		if result.signal == agents.MainServed {
			dish := e.stock.deliverMain(worker.job.dishID)
			student.result.Meal.Main = &dish
			dishes = []string{dish.ID}
			e.scoreMeal(student)
		}
		e.emit(result.signal, result.studentID, worker.result.Staff.ID, student.result.Area, student.result.Area, dishes...)
		worker.job = nil
	}
}

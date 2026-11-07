package engine

import (
	agents "simuresto/SimuResto/Agents"
)

// 只有队首可以前进；下一区域满员时，学生留在当前区域。
func (e *engine) enter(student *studentState, area agents.Area) {
	id, from := student.result.Student.ID, student.result.Area
	if !e.queues[area].Enqueue(id) {
		panic("simulation: transfer into a full queue")
	}
	student.result.Area = area
	student.result.Path = append(student.result.Path, area)
	student.started, student.ready, student.elapsed = false, false, 0
	e.stats.peakOccupancy[area] = max(e.stats.peakOccupancy[area], e.queues[area].Len())
	e.emit(agents.Entered, id, 0, from, area)
}

func (e *engine) admit() {
	for e.nextArrivalIndex < len(e.arrivalStudent) && e.arrivalStudent[e.nextArrivalIndex].result.Student.ArrivalTime <= e.now {
		student := e.arrivalStudent[e.nextArrivalIndex]
		e.nextArrivalIndex++
		student.result.ArrivalTime = e.now
		e.emit(agents.Arrived, student.result.Student.ID, 0, agents.NoArea, agents.NoArea)
		area := agents.Outside
		if e.queues[agents.Outside].Len() == 0 && e.queues[agents.Inside].Len() < e.queues[agents.Inside].Capacity() {
			area = agents.Inside
		}
		if e.queues[area].Len() == e.queues[area].Capacity() {
			student.result.Status = agents.Rejected
			student.result.ExitTime = e.now
			e.stats.finishedStudentCount++
			e.emit(agents.QueueFull, student.result.Student.ID, 0, agents.NoArea, agents.NoArea)
			continue
		}
		student.result.Status = agents.Active
		e.enter(student, area)
		// 立即推进队列，让同时到达的学生使用所有空余位置，
		// 并始终优先接纳已经在室外排队的学生。
		e.moveForward()
	}
}

// 往前走，前进一步
func (e *engine) moveForward() {
	route := agents.ServiceRoute()
	for {
		changed := false
		for i := len(route) - 1; i >= 0; i-- {
			area, queue := route[i], e.queues[route[i]]
			for {
				id, ok := queue.Front()
				if !ok {
					break
				}

				student := e.ID2Student[id]
				if !area.IsWaitingArea() && !student.ready {
					break
				}

				if area == agents.Payment {
					queue.Dequeue()
					student.result.Area = agents.NoArea
					student.result.Status = agents.Completed
					student.result.ExitTime = e.now
					e.stats.finishedStudentCount++
					e.emit(agents.Finished, id, 0, agents.Payment, agents.NoArea)
				} else {
					next := route[i+1]
					if e.queues[next].Len() == e.queues[next].Capacity() {
						break //full
					}
					queue.Dequeue()
					e.enter(student, next)
				}
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

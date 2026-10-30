package simulation

// 只有队首可以前进；下一区域满员时，学生留在当前区域。
func (e *engine) enter(student *studentState, area Area) {
	id, from := student.result.Student.ID, student.result.Area
	if !e.queues[area].Enqueue(id) {
		panic("simulation: transfer into a full queue")
	}
	student.result.Area = area
	student.result.Path = append(student.result.Path, area)
	student.started, student.ready, student.elapsed = false, false, 0
	e.peaks[area] = max(e.peaks[area], e.queues[area].Len())
	e.emit(Entered, id, 0, from, area)
}

func (e *engine) admit() {
	for e.next < len(e.arrivals) && e.arrivals[e.next].result.Student.ArrivalTime <= e.now {
		student := e.arrivals[e.next]
		e.next++
		student.result.ArrivalTime = e.now
		e.emit(Arrived, student.result.Student.ID, 0, NoArea, NoArea)
		area := Outside
		if e.queues[Outside].Len() == 0 && e.queues[Inside].Len() < e.queues[Inside].Capacity() {
			area = Inside
		}
		if e.queues[area].Len() == e.queues[area].Capacity() {
			student.result.Status = Rejected
			student.result.ExitTime = e.now
			e.finished++
			e.emit(QueueFull, student.result.Student.ID, 0, NoArea, NoArea)
			continue
		}
		student.result.Status = Active
		e.enter(student, area)
		// 立即推进队列，让同时到达的学生使用所有空余位置，
		// 并始终优先接纳已经在室外排队的学生。
		e.moveForward()
	}
}

func (e *engine) moveForward() {
	route := ServiceRoute()
	for {
		changed := false
		for i := len(route) - 1; i >= 0; i-- {
			area, queue := route[i], e.queues[route[i]]
			for {
				id, ok := queue.Front()
				if !ok {
					break
				}
				student := e.byID[id]
				if !area.IsWaitingArea() && !student.ready {
					break
				}
				if area == Payment {
					queue.Dequeue()
					student.result.Area = NoArea
					student.result.Status = Completed
					student.result.ExitTime = e.now
					e.finished++
					e.emit(Finished, id, 0, Payment, NoArea)
				} else {
					next := route[i+1]
					if e.queues[next].Len() == e.queues[next].Capacity() {
						break
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

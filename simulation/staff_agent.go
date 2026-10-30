package simulation

import "time"

type assignment struct {
	studentID int
	duration  time.Duration
}

type workTick struct {
	start   *assignment
	elapsed time.Duration
}

type workResult struct {
	signal    Signal
	studentID int
	busy      time.Duration
}

type staffAgent struct {
	in  chan workTick
	out chan workResult
}

// 工作人员在自己的 goroutine 中维护当前任务和剩余服务时间。
func runStaff(staff Staff, agent staffAgent) {
	var current assignment
	var remaining time.Duration
	for tick := range agent.in {
		if tick.start != nil {
			current = *tick.start
			remaining = current.duration
		}
		busy := min(tick.elapsed, remaining)
		remaining -= busy
		signal := Progress
		if remaining == 0 {
			signal = MainServed
			if staff.Role == Cashier {
				signal = Paid
			}
		}
		agent.out <- workResult{signal: signal, studentID: current.studentID, busy: busy}
	}
}

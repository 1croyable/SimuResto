package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type assignment struct {
	studentID int
}

type workTick struct {
	start   *assignment
	elapsed time.Duration
}

type workResult struct {
	signal    agents.Signal
	studentID int
	busy      time.Duration
}

type staffAgent struct {
	in  chan workTick
	out chan workResult
}

// 工作人员在自己的 goroutine 中维护当前任务和剩余服务时间。
func runStaff(staff agents.Staff, agent staffAgent) {
	minimum, maximum := environment.MinMainServiceTime, environment.MaxMainServiceTime
	if staff.Role == agents.Cashier {
		minimum, maximum = environment.MinPaymentTime, environment.MaxPaymentTime
	}
	var current assignment
	var remaining time.Duration
	for tick := range agent.in {
		if tick.start != nil {
			current = *tick.start
			// 新任务只抽取一次耗时，后续时钟更新继续扣减剩余时间。
			remaining = randomServiceTime(minimum, maximum)
		}
		busy := min(tick.elapsed, remaining)
		remaining -= busy
		signal := agents.Progress
		if remaining == 0 {
			signal = agents.MainServed
			if staff.Role == agents.Cashier {
				signal = agents.Paid
			}
		}
		agent.out <- workResult{signal: signal, studentID: current.studentID, busy: busy}
	}
}

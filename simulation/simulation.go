package simulation

import (
	"context"
	"errors"
	"time"
)

var ErrTimeLimit = errors.New("simulation reached its maximum duration")

// Run 为每名学生和工作人员启动一个 goroutine。Tick 表示模拟步长，
// TickInterval 用于控制播放速度。同样的输入产生相同的报告，
// 不受 goroutine 调度顺序影响。取消运行或达到时限时，
// 返回部分报告、归还预留库存，并等待所有智能体退出。
func Run(ctx context.Context, config Config, students []Student) (Report, error) {
	if err := config.validate(students); err != nil {
		return Report{}, err
	}
	e := newEngine(config, students)
	defer e.closeAgents()
	for {
		if err := ctx.Err(); err != nil {
			e.stop()
			return e.report(), err
		}
		e.moveForward()
		e.admit()
		if e.finished == len(e.students) {
			return e.report(), nil
		}
		if e.now >= config.MaxDuration {
			e.stop()
			return e.report(), ErrTimeLimit
		}

		e.askStudents(allStudents)
		e.startSelfService()
		e.assignStaff()
		step := min(config.Tick, config.MaxDuration-e.now)
		e.countWaiting(step)
		e.now += step
		e.advanceSelfService(step)
		e.advanceStaffService(step)
		// 先让排队学生决定是否离队，再把新腾出的位置
		// 分配给继续等待的学生。
		e.askStudents(waitingStudents)
		e.moveForward()
		if e.finished == len(e.students) {
			return e.report(), nil
		}

		if config.TickInterval > 0 && e.now < config.MaxDuration {
			timer := time.NewTimer(config.TickInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				e.stop()
				return e.report(), ctx.Err()
			case <-timer.C:
			}
		}
	}
}

package simulation

import "context"

// Session 让命令行和图形前端共用同一套模拟。调用方串行推进，智能体仍各自运行在 goroutine 中。
// 前端暂停时不调用 Step，因此耐心、等待和服务时间也会一起暂停。
type Session struct {
	engine *engine
	done   bool
	err    error
}

func NewSession(config Config, students []Student) (*Session, error) {
	if err := config.validate(students); err != nil {
		return nil, err
	}
	return &Session{engine: newEngine(config, students)}, nil
}

// Step 推进一个逻辑时间步，返回是否结束。不可从多个 goroutine 同时调用。
func (s *Session) Step(ctx context.Context) (bool, error) {
	if s.done {
		return true, s.err
	}
	e := s.engine
	if err := ctx.Err(); err != nil {
		e.stop()
		return s.finish(err)
	}
	e.moveForward()
	e.admit()
	if e.finished == len(e.students) {
		return s.finish(nil)
	}
	if e.now >= e.config.MaxDuration {
		e.stop()
		return s.finish(ErrTimeLimit)
	}

	e.askStudents(allStudents)
	// 离队腾出的空位立即交给后方学生，再开始服务并统计真实等待。
	e.moveForward()
	e.startSelfService()
	e.assignStaff()
	step := min(e.config.Tick, e.config.MaxDuration-e.now)
	e.countWaiting(step)
	e.now += step
	e.advanceSelfService(step)
	e.advanceStaffService(step)
	e.askStudents(waitingStudents)
	e.moveForward()
	if e.finished == len(e.students) {
		return s.finish(nil)
	}
	return false, nil
}

func (s *Session) finish(err error) (bool, error) {
	s.done, s.err = true, err
	s.engine.closeAgents()
	return true, err
}

// Close 用于关闭窗口或重新开始；归还未交付的库存并等待所有智能体退出。
// 已结束或已关闭的会话可以重复关闭。
func (s *Session) Close() {
	if !s.done {
		s.engine.stop()
		s.finish(context.Canceled)
	}
}

// Report 返回独立副本，调用方修改它不会影响模拟。
func (s *Session) Report() Report { return cloneReport(s.engine.report()) }

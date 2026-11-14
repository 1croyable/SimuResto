package engine

import (
	"context"
	"fmt"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type Game struct {
	engine *engine
	done   bool
	err    error
}

func NewGame(config environment.Config, students []agents.Student) (*Game, error) {
	if err := config.Validate(students); err != nil {
		return nil, err
	}
	return &Game{engine: newEngine(config, students)}, nil
}

// SetCashiers 调整开放窗口数量，与 Step 一样由调用方串行调用。
// 关闭的窗口会完成手中的付款，再停止接单；本轮时间和统计不重置。
func (g *Game) SetCashiers(count int) error {
	if count < 1 || count > environment.MaxCashiers {
		return fmt.Errorf("收银窗口数量必须为 1 或 2")
	}
	if g.done {
		return fmt.Errorf("本轮模拟已结束")
	}
	queue := g.engine.queues[agents.Payment]
	if err := queue.Resize(max(count, queue.Len())); err != nil {
		return err
	}
	g.engine.config.Cashiers = count
	return nil
}

// Step 推进一个逻辑时间步，返回是否结束。不可从多个 goroutine 同时调用。
func (g *Game) Step(ctx context.Context) (bool, error) {
	if g.done {
		return true, g.err
	}
	e := g.engine
	if err := ctx.Err(); err != nil {
		e.stop()
		return g.finish(err)
	}
	e.moveForward()
	e.admit()
	if e.stats.finishedStudentCount == len(e.students) {
		return g.finish(nil)
	}
	if e.now >= e.config.MaxDuration {
		e.stop()
		return g.finish(ErrTimeLimit)
	}

	e.askStudents(allStudents)
	// 离队腾出的空位立即交给后方学生，再开始服务并统计真实等待。
	e.moveForward()
	e.startSelfService()
	e.assignStaff()
	step := min(e.config.Tick, e.config.MaxDuration-e.now)
	e.countWaiting(step)
	e.now += step
	e.stepSelfService(step)
	e.stepStaffService(step)
	e.askStudents(waitingStudents) //室外学生是否放弃排队
	e.moveForward()
	if e.stats.finishedStudentCount == len(e.students) {
		return g.finish(nil)
	}
	return false, nil
}

func (g *Game) finish(err error) (bool, error) {
	g.done, g.err = true, err
	g.engine.closeAgents()
	return true, err
}

// Close 用于关闭窗口或重新开始；归还未交付的库存并等待所有智能体退出。
// 已结束或已关闭的模拟可以重复关闭。
func (g *Game) Close() {
	if !g.done {
		g.engine.stop()
		g.finish(context.Canceled)
	}
}

// Report 返回独立副本，调用方修改它不会影响模拟。
func (g *Game) Report() Report { return cloneReport(g.engine.report()) }

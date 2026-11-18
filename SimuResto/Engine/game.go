package engine

import (
	"fmt"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type Game struct {
	engine   *engine
	World    *environment.Environment
	snapshot Snapshot
	done     bool
	err      error
}

func NewGame(world *environment.Environment) *Game {
	g := &Game{
		World:  world,
		engine: newEngine(),
	}

	g.snapshot = buildSnapshot(world, false, Snapshot{})

	return g
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

func (g *Game) Step(dt time.Duration) {
	world := g.World
	engine := g.engine

	if dt <= 0 || g.done {
		return
	}

	world.TimePassed += dt

	arrivedStudents := make(chan *agents.StudentState, len(world.StudentStates))
	for studentIndex := range world.StudentStates {
		student := &world.StudentStates[studentIndex]
		if student.Status == agents.Scheduled && student.Student.ArrivalTime <= world.TimePassed {
			arrivedStudents <- student
		}
	}
	close(arrivedStudents)

	// admit arrived students while there is space in the queue
	engine.admitStudents(world, arrivedStudents)

	if n := len(world.Queue.IDs()); n > world.Statistics.MaxQueueLength {
		world.Statistics.MaxQueueLength = n
	}

	// advance student state
	for studentIndex := range world.StudentStates {
		student := &world.StudentStates[studentIndex]
		advanceStudentAfterPayment(world, student, dt)
	}

	// students still in line move, collect food, pay, or leave the queue
	g.engine.advanceQueue(world, dt)

	// fill places freed by payment or abandonment during this step
	engine.admitStudents(world, arrivedStudents)

	g.snapshot = buildSnapshot(g.World, g.done, g.snapshot)
}

func (g *Game) finish(err error) (bool, error) {
	if g.done {
		return true, g.err
	}

	g.done = true
	g.err = err
	g.engine.closeAgents()

	g.snapshot = buildSnapshot(g.World, true, g.snapshot)
	return true, err
}

func (g *Game) Close() {
	g.finish(nil)
}

func (g *Game) Report() Report {
	return g.Snapshot().Report
}

package engine

import (
	"time"

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

	g.engine.UpdateOperatingPhase(world)

	if err := g.engine.initArrivals(world); err != nil {
		panic(err)
	}

	g.snapshot = buildSnapshot(world, false, Snapshot{})

	return g
}

func (g *Game) Step(dt time.Duration) {
	world := g.World
	engine := g.engine
	setting := world.Settings

	if dt <= 0 || g.done {
		return
	}

	// Current restaurant clock before this tick.
	from := world.Settings.StartTime + world.TimePassed

	world.TimePassed += dt
	engine.UpdateOperatingPhase(world)

	// Current restaurant clock after this tick.
	to := world.Settings.StartTime + world.TimePassed

	newcomers := engine.spawnArrivals(world, from, to)
	engine.admitStudents(world, newcomers)

	lunchClosed := from < setting.Lunch.Close && to >= setting.Lunch.Close
	dinnerClosed := from < setting.Dinner.Close && to >= setting.Dinner.Close

	if lunchClosed || dinnerClosed {
		engine.removeOutsideStudents(world)
	}

	if n := len(world.Queue.IDs()); n > world.Statistics.MaxQueueLength {
		world.Statistics.MaxQueueLength = n
	}

	// advance student state
	for studentIndex := range world.StudentStates {
		student := &world.StudentStates[studentIndex]
		advanceStudentOutsideQueue(world, student, dt)
	}

	// students still in line move, collect food, pay, or leave the queue
	g.engine.advanceQueue(world, dt)

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

func (g *Game) SetSimulationSpeed(speed float64, world *environment.Environment) {
	if speed > 0 {
		world.SimulationSpeed = speed
	}
}

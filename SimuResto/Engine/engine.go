package engine

import (
	agents "simuresto/SimuResto/Agents"
	randomness "simuresto/SimuResto/Agents/Randomness"
	environment "simuresto/SimuResto/Environment"
	"sync"
	"time"
)

// The engine coordinates agent execution.
// Environment holds mutable simulation state, while dining holds the fixed map.
type engine struct {
	StudentAgents map[int]*StudentAgent
	StaffAgents   map[int]*StaffAgent

	agentWG   sync.WaitGroup
	closeOnce sync.Once

	arrivals  *randomness.ArrivalProcess
	generator *agents.StudentGenerator
	dining    DiningLayout
}

func newEngine() *engine {
	return &engine{
		StudentAgents: make(map[int]*StudentAgent),
		StaffAgents:   make(map[int]*StaffAgent),
	}
}

func (e *engine) closeAgents() {
	e.closeOnce.Do(func() {
		for _, agent := range e.StudentAgents {
			close(agent.in)
		}

		for _, agent := range e.StaffAgents {
			close(agent.in)
		}

		e.agentWG.Wait()
	})
}

func (e *engine) CurrentTime(world *environment.Environment) time.Duration {
	return world.Settings.StartTime + world.TimePassed
}

func (e *engine) UpdateOperatingPhase(world *environment.Environment) {
	now := e.CurrentTime(world)
	s := world.Settings

	switch {
	case now < s.Lunch.Open:
		world.OperatingPhase = environment.OperatingBreak

	case now < s.Lunch.Close:
		world.OperatingPhase = environment.OperatingOpen

	case now < s.Dinner.Open:
		world.OperatingPhase = environment.OperatingBreak

	case now < s.Dinner.Close:
		world.OperatingPhase = environment.OperatingOpen

	default:
		world.OperatingPhase = environment.OperatingClosing
	}
}

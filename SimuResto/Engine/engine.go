package engine

import "sync"

// The engine coordinates agent execution.
// All simulation states are maintained by Environment.
type engine struct {
	StudentAgents map[int]*StudentAgent
	StaffAgents   map[int]*StaffAgent

	agentWG   sync.WaitGroup
	closeOnce sync.Once
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

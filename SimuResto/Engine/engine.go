package engine

import (
	"sync"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

// Represents a task assigned to a staff member.
type job struct {
	studentID int
	dishID    string
	fresh     bool
}

// Runtime state of a staff agent.
// Staff goroutine integration will be refined later.
type staffState struct {
	result StaffResult
	agent  staffAgent
	job    *job
}

// The engine coordinates agent execution.
// Simulation data is maintained by Environment.
type engine struct {
	StudentAgents map[int]*StudentAgent
	staff         []*staffState
	// Tracks all student and staff goroutines.
	agentWG sync.WaitGroup

	// Prevents closing agent channels multiple times.
	closeOnce sync.Once
}

// Initialize agent runtime resources.
func newEngine() *engine {
	e := &engine{
		StudentAgents: make(map[int]*StudentAgent),
	}

	// Initialize staff agents.
	// Keep the teammate's original implementation for now.
	for i := 0; i < environment.Servers+environment.MaxCashiers; i++ {
		staff := agents.Staff{
			ID:   i + 1,
			Role: agents.Server,
		}

		if i >= environment.Servers {
			staff.Role = agents.Cashier
		}

		state := &staffState{
			result: StaffResult{Staff: staff},
			agent: staffAgent{
				in:  make(chan workTick),
				out: make(chan workResult),
			},
		}

		e.staff = append(e.staff, state)

		e.agentWG.Add(1)
		go func() {
			defer e.agentWG.Done()
			runStaff(staff, state.agent)
		}()
	}

	return e
}

func (e *engine) closeAgents() {
	e.closeOnce.Do(func() {
		for _, agent := range e.StudentAgents {
			close(agent.in)
		}

		for _, staff := range e.staff {
			close(staff.agent.in)
		}

		e.agentWG.Wait()
	})
}

package engine

import (
	"sync"

	agents "simuresto/SimuResto/Agents"
)

type StaffAgent struct {
	in  chan agents.StaffObservation
	out chan agents.StaffDecision
}

func startStaffAgent(staff *agents.StaffState, wg *sync.WaitGroup) *StaffAgent {
	agent := &StaffAgent{
		in:  make(chan agents.StaffObservation),
		out: make(chan agents.StaffDecision),
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		for observation := range agent.in {
			decision := staff.Decide(observation)
			agent.out <- decision
		}
	}()

	return agent
}

func (agent *StaffAgent) ask(observation agents.StaffObservation) agents.StaffDecision {
	agent.in <- observation
	return <-agent.out
}

func (e *engine) requestStaffDecision(staff *agents.StaffState, observation agents.StaffObservation) agents.StaffDecision {
	id := staff.Staff.ID

	agent := e.StaffAgents[id]

	if agent == nil {
		agent = startStaffAgent(staff, &e.agentWG)
		e.StaffAgents[id] = agent
	}

	return agent.ask(observation)
}

func (e *engine) startCounterService(staff *agents.StaffState, studentID, queueLength int) bool {
	decision := e.requestStaffDecision(staff, agents.StaffObservation{
		NextStudentID: studentID,
		QueueLength:   queueLength,
		Open:          true, // An already assigned customer can be served after closing.
	})
	if decision.AcceptedStudentID != studentID {
		return false
	}
	staff.StudentID = studentID
	staff.ServiceDuration = decision.RemainingTime
	staff.ServiceTimeLeft = decision.RemainingTime
	return true
}

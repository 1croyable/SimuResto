package engine

import (
	"sync"

	agents "simuresto/SimuResto/Agents"
)

type StudentAgent struct {
	in  chan agents.StudentObservation
	out chan agents.StudentDecision
}

func startStudentAgent(student *agents.StudentState, wg *sync.WaitGroup) *StudentAgent {
	agent := &StudentAgent{
		in:  make(chan agents.StudentObservation),
		out: make(chan agents.StudentDecision),
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		for observation := range agent.in {
			decision := student.Decide(observation)
			agent.out <- decision
		}
	}()

	return agent
}

func (agent *StudentAgent) ask(observation agents.StudentObservation) agents.StudentDecision {
	agent.in <- observation
	return <-agent.out
}

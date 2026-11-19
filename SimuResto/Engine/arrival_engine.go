package engine

import (
	"fmt"
	"math"
	"time"

	agents "simuresto/SimuResto/Agents"
	randomness "simuresto/SimuResto/Agents/Randomness"
	environment "simuresto/SimuResto/Environment"
)

func (e *engine) initArrivals(world *environment.Environment) error {
	s := world.Settings
	if s.StudentCount < 0 || s.DinnerShare < 0 || s.DinnerShare > 1 || math.IsNaN(s.DinnerShare) {
		return fmt.Errorf("invalid student count or dinner share")
	}
	dinner := int(math.Round(float64(s.StudentCount) * s.DinnerShare))
	makeWindow := func(p environment.ArrivalPeriod, count int) randomness.ArrivalWindow {
		return randomness.ArrivalWindow{
			Count:           count,
			Open:            p.Open,
			Close:           p.Close,
			Peak:            p.Peak,
			EarlyLead:       p.EarlyLead,
			SigmaLeft:       p.SigmaLeft,
			SigmaRight:      p.SigmaRight,
			EarlyRatio:      p.EarlyRatio,
			UniformRatio:    p.UniformRatio,
			StopBeforeClose: p.StopBeforeClose,
		}
	}
	windows := [2]randomness.ArrivalWindow{
		makeWindow(s.Lunch, s.StudentCount-dinner), makeWindow(s.Dinner, dinner),
	}
	process, err := randomness.NewArrivalProcess(windows, s.StartTime, s.Seed)
	if err != nil {
		return err
	}
	e.arrivals = process
	e.generator = agents.NewStudentGenerator(s.Seed, s.BasePatience)
	return nil
}

// Create only the students who actually arrive during this update.
func (e *engine) spawnArrivals(world *environment.Environment, from, to time.Duration) []*agents.StudentState {
	counts := e.arrivals.Next(from, to)
	newcomers := make([]*agents.StudentState, 0, counts[0]+counts[1])
	for _, count := range counts {
		for k := 0; k < count; k++ {
			id := len(world.StudentStates) + 1
			data := e.generator.Generate(id, world.TimePassed)
			world.StudentStates = append(world.StudentStates, agents.NewStudentState(data, world.Settings.Seed))
			student := &world.StudentStates[len(world.StudentStates)-1]
			world.StudentsByID[id] = student

			newcomers = append(newcomers, student)
		}
	}
	return newcomers
}

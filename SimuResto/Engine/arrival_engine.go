package engine

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"time"

	agents "simuresto/SimuResto/Agents"
	randomness "simuresto/SimuResto/Agents/Randomness"
	environment "simuresto/SimuResto/Environment"
)

const (
	companionLeaderProbability = 0.1
	maxPendingCompanionLeaders = 2
	maxCompanions              = 3
	minCompanionDelay          = 20 * time.Second
	maxCompanionDelay          = 90 * time.Second
)

type companionArrival struct {
	due       time.Duration
	expires   time.Duration
	leaderID  int
	remaining int
}

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
	e.companionRandom = rand.New(rand.NewSource(s.Seed))
	return nil
}

// Create only the students who actually arrive during this update.
func (e *engine) spawnArrivals(world *environment.Environment, from, to time.Duration) []*agents.StudentState {
	counts := e.arrivals.Next(from, to)
	newcomers := make([]*agents.StudentState, 0, counts[0]+counts[1])
	closing := [2]time.Duration{world.Settings.Lunch.Close, world.Settings.Dinner.Close}
	for meal, count := range counts { //meal: 0=lunch 1=dinner
		for range count {
			companion := e.nextCompanion(world.TimePassed) // Check if the students is attached to his friend.
			student := e.spawnStudent(world, companion)
			if companion.Role == agents.NoCompanion {
				e.planCompanions(student, world.TimePassed, closing[meal]-world.Settings.StartTime)
			}
			newcomers = append(newcomers, student)
		}
	}
	return newcomers
}

func (e *engine) nextCompanion(now time.Duration) agents.Companion {
	remaining := e.companionArrivals[:0] //empty list
	for _, group := range e.companionArrivals {
		if group.expires > now {
			remaining = append(remaining, group)
		}
	} //delete expired
	e.companionArrivals = remaining
	if len(e.companionArrivals) == 0 || e.companionArrivals[0].due > now {
		return agents.Companion{}
	}

	//alloc a companion
	group := &e.companionArrivals[0]
	companion := agents.Companion{Role: agents.InvitedCompanion, LeaderID: group.leaderID}
	group.remaining--
	if group.remaining == 0 {
		e.companionArrivals = e.companionArrivals[1:]
	}
	return companion
}

func (e *engine) planCompanions(student *agents.StudentState, now, closing time.Duration) {
	if now >= closing || len(e.companionArrivals) >= maxPendingCompanionLeaders ||
		e.companionRandom.Float64() >= companionLeaderProbability {
		return
	}
	group := companionArrival{
		due:       now,
		expires:   closing,
		leaderID:  student.Student.ID,
		remaining: 1 + e.companionRandom.Intn(maxCompanions),
	}
	if e.companionRandom.Intn(2) == 0 {
		student.Student.Companion.Role = agents.TogetherLeader
	} else {
		student.Student.Companion.Role = agents.QueueJumpLeader
		delay := minCompanionDelay + time.Duration(e.companionRandom.Int63n(int64(maxCompanionDelay-minCompanionDelay)+1))
		group.due += delay
	}
	index := len(e.companionArrivals)
	for index > 0 && e.companionArrivals[index-1].due > group.due {
		index--
	}
	e.companionArrivals = slices.Insert(e.companionArrivals, index, group)
}

func (e *engine) spawnStudent(world *environment.Environment, companion agents.Companion) *agents.StudentState {
	id := len(world.StudentStates) + 1
	data := e.generator.Generate(id, world.TimePassed)
	data.Companion = companion
	world.StudentStates = append(world.StudentStates, agents.NewStudentState(data, world.Settings.Seed))
	student := &world.StudentStates[len(world.StudentStates)-1]
	world.StudentsByID[id] = student
	return student
}

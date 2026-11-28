package engine

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	agents "simuresto/SimuResto/Agents"
	randomness "simuresto/SimuResto/Agents/Randomness"
	environment "simuresto/SimuResto/Environment"
)

const (
	companionLeaderProbability = 0.1
	maxPendingQueueJumpLeaders = 2
	maxCompanions              = 3
	minCompanionDelay          = 20 * time.Second
	maxCompanionDelay          = 90 * time.Second
)

var companionPalette = [...]color.RGBA{
	{R: 209, G: 55, B: 57, A: 255},  // Red
	{R: 231, G: 190, B: 25, A: 255}, // Yellow
	{R: 139, G: 74, B: 184, A: 255}, // Purple
	{R: 133, G: 85, B: 54, A: 255},  // Brown
	{R: 99, G: 195, B: 240, A: 255}, // Sky blue
}

type companionArrival struct {
	due       time.Duration
	expires   time.Duration
	leaderID  int // Zero while a group is waiting to arrive together
	remaining int
	buffered  int
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
	process, err := randomness.NewArrivalProcess(windows, s.StartTime)
	if err != nil {
		return err
	}
	e.arrivals = process
	e.generator = agents.NewStudentGenerator(s.BasePatience)
	return nil
}

// Create only the students who actually arrive during this update.
func (e *engine) spawnArrivals(world *environment.Environment, from, to time.Duration) []*agents.StudentState {
	counts := e.arrivals.Next(from, to)
	newcomers := make([]*agents.StudentState, 0, counts[0]+counts[1])
	periods := [2]environment.ArrivalPeriod{world.Settings.Lunch, world.Settings.Dinner}
	for meal, count := range counts {
		period := periods[meal]
		closing := period.Close - world.Settings.StartTime
		cutoff := closing - period.StopBeforeClose
		windowEnded := to-world.Settings.StartTime >= cutoff
		for {
			if len(e.companionArrivals) > 0 &&
				e.companionArrivals[0].leaderID == 0 && e.companionArrivals[0].expires == cutoff {
				group := &e.companionArrivals[0]
				// Reserve arrival slots until the group can enter together.
				added := min(count, group.remaining)
				group.buffered += added
				group.remaining -= added
				count -= added
				if group.remaining > 0 && !windowEnded {
					break
				}
				leader := e.spawnStudent(world, agents.Companion{Role: agents.TogetherLeader})
				newcomers = append(newcomers, leader)
				for i := 1; i < group.buffered; i++ {
					companion := agents.Companion{Role: agents.InvitedCompanion, LeaderID: leader.Student.ID}
					newcomers = append(newcomers, e.spawnStudent(world, companion))
				}
				e.companionArrivals = e.companionArrivals[1:]
				continue
			}
			if count == 0 {
				break
			}
			companion := e.nextCompanion(world.TimePassed)
			if companion.Role == agents.NoCompanion {
				companion.Role = e.planCompanions(len(world.StudentStates)+1, world.TimePassed, closing, cutoff)
				if companion.Role == agents.TogetherLeader {
					continue
				}
			}
			newcomers = append(newcomers, e.spawnStudent(world, companion))
			count--
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

func (e *engine) planCompanions(leaderID int, now, closing, cutoff time.Duration) agents.CompanionRole {
	if now >= closing || rand.Float64() >= companionLeaderProbability {
		return agents.NoCompanion
	}
	companions := 1 + rand.IntN(maxCompanions)
	if rand.IntN(2) == 0 {
		// Give the group consecutive slots, including one for its leader.
		e.companionArrivals = slices.Insert(e.companionArrivals, 0, companionArrival{
			expires:   cutoff,
			remaining: 1 + companions,
		})
		return agents.TogetherLeader
	}

	pendingQueueJumpLeaders := 0
	for _, group := range e.companionArrivals {
		if group.leaderID != 0 && group.expires > now {
			pendingQueueJumpLeaders++
		}
	}
	if pendingQueueJumpLeaders >= maxPendingQueueJumpLeaders {
		return agents.NoCompanion
	}

	delay := minCompanionDelay + time.Duration(rand.Int64N(int64(maxCompanionDelay-minCompanionDelay)+1))
	group := companionArrival{
		due:       now + delay,
		expires:   closing,
		leaderID:  leaderID,
		remaining: companions,
	}
	index := len(e.companionArrivals)
	for index > 0 && e.companionArrivals[index-1].due > group.due {
		index--
	}
	e.companionArrivals = slices.Insert(e.companionArrivals, index, group)
	return agents.QueueJumpLeader
}

func (e *engine) spawnStudent(world *environment.Environment, companion agents.Companion) *agents.StudentState {
	id := len(world.StudentStates) + 1
	data := e.generator.Generate(id, world.TimePassed)
	switch companion.Role {
	case agents.TogetherLeader, agents.QueueJumpLeader:
		companion.Color = companionPalette[rand.IntN(len(companionPalette))]
	case agents.InvitedCompanion:
		if leader := world.StudentsByID[companion.LeaderID]; leader != nil {
			companion.Color = leader.Student.Companion.Color
		}
	}
	data.Companion = companion
	world.StudentStates = append(world.StudentStates, agents.NewStudentState(data))
	student := &world.StudentStates[len(world.StudentStates)-1]
	world.StudentsByID[id] = student
	return student
}

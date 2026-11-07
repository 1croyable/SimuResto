package agents

import (
	"math"
	"math/rand"
	randomness "simuresto/SimuResto/Agents/Randomness"
	"time"
)

type StudentData struct {
	ID             int
	ArrivalTime    time.Duration
	Patience       time.Duration
	StartersWanted int
	DessertsWanted int
}

type DishOption struct {
	ID string
}

type StudentObservation struct {
	CanLeave    bool
	PeopleAhead int           // People ahead, including anyone currently being served.
	Waiting     time.Duration // Total time spent
	Elapsed     time.Duration // Time spent waiting during this update.
	Dishes      []DishOption
}

type StudentDecision struct {
	Leave  bool
	DishID string
}

// Walk follows the route at walkingSpeed map units per second and returns the unused time.
func (studentStatus *StudentState) Walk(route []Point, walkingSpeed float64, dt time.Duration) time.Duration {
	for studentStatus.NextWaypoint < len(route) {
		target := route[studentStatus.NextWaypoint]
		dx, dy := target.X-studentStatus.Position.X, target.Y-studentStatus.Position.Y
		distance := math.Hypot(dx, dy)
		// total time needed to reach the next route point
		travelTime := time.Duration(math.Round(distance / walkingSpeed * float64(time.Second)))

		if travelTime > dt {
			// fraction is the share of the remaining segment covered this step, used to calculate the new position
			fraction := walkingSpeed * dt.Seconds() / distance

			// Move a part of the way toward the next point
			studentStatus.Position.X += dx * fraction
			studentStatus.Position.Y += dy * fraction
			return 0
		}

		studentStatus.Position = target
		studentStatus.NextWaypoint++
		dt -= travelTime
	}

	return dt
}

func (studentStatus *StudentState) chooseDish(dishes []DishOption) string {
	if len(dishes) == 0 {
		return ""
	}

	randomDishIndex := studentStatus.randomGenerator.Intn(len(dishes))

	return dishes[randomDishIndex].ID
}

func (studentStatus *StudentState) Decide(state StudentObservation) StudentDecision {
	if state.CanLeave && state.Elapsed > 0 {
		probabilityPerSecond := randomness.LeaveProbability(studentStatus.Student.Patience, state.Waiting, state.PeopleAhead)

		// P(leave within t seconds) = 1 - (1 - P(leave within one second))^t.
		probabilityThisUpdate := -math.Expm1(state.Elapsed.Seconds() * math.Log1p(-probabilityPerSecond))

		if studentStatus.randomGenerator.Float64() < probabilityThisUpdate {
			return StudentDecision{Leave: true}
		}
	}

	return StudentDecision{DishID: studentStatus.chooseDish(state.Dishes)}
}

func NewStudentState(student StudentData, seed int64) StudentState {
	studentSeed := int64(uint64(seed) ^ (uint64(student.ID) * 0x9e3779b97f4a7c15))
	return StudentState{
		Student:         student,
		Status:          Scheduled,
		randomGenerator: rand.New(rand.NewSource(studentSeed)),
	}
}

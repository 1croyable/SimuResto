package agents

import (
	"math"
	"math/rand"
	randomness "simuresto/SimuResto/Agents/Randomness"
	"time"
)

type StudentData struct {
	ID          int
	ArrivalTime time.Duration
	Patience    time.Duration

	// preference of the dishes
	ColorWeight   float64
	TasteWeight   float64
	SatietyWeight float64
	Vegetarian    bool
}

type StudentObservation struct {
	CanLeave       bool
	PeopleAhead    int           // People ahead, including anyone currently being served.
	Waiting        time.Duration // Total time spent
	Elapsed        time.Duration // Time spent waiting during this update.
	Dishes         []Dish
	Temperature    float64
	ChooseStarters bool // Request starter selection.
}

type StudentDecision struct {
	Leave      bool
	DishID     string
	StarterIDs []string
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

func (studentStatus *StudentState) Decide(state StudentObservation) StudentDecision {
	// choose starters
	if state.ChooseStarters {
		return StudentDecision{
			StarterIDs: studentStatus.PlanStarters(
				state.Dishes,
				state.Temperature,
			),
		}
	}

	// choose to give up
	if state.CanLeave && state.Elapsed > 0 {
		probabilityPerSecond := randomness.LeaveProbability(studentStatus.Student.Patience, state.Waiting, state.PeopleAhead)

		// P(leave within t seconds) = 1 - (1 - P(leave within one second))^t.
		probabilityThisUpdate := -math.Expm1(state.Elapsed.Seconds() * math.Log1p(-probabilityPerSecond))

		if studentStatus.randomGenerator.Float64() < probabilityThisUpdate {
			return StudentDecision{Leave: true}
		}
	}

	// choose a main course if dishes are available
	if len(state.Dishes) > 0 {
		return StudentDecision{
			DishID: studentStatus.chooseDish(state.Dishes, state.Temperature),
		}
	}

	return StudentDecision{}
}

func NewStudentState(student StudentData, seed int64) StudentState {
	studentSeed := int64(uint64(seed) ^ (uint64(student.ID) * 0x9e3779b97f4a7c15))
	return StudentState{
		Student:         student,
		Status:          Scheduled,
		randomGenerator: rand.New(rand.NewSource(studentSeed)),
		CashierIndex:    -1,
	}
}

func (studentStatus *StudentState) PlanStarters(menu []Dish, temperature float64) []string {
	planned := make([]string, 0, 2)

	portions := make([]int, len(menu))
	for i, dish := range menu {
		portions[i] = dish.Portions
	}

	colorWeight := studentStatus.Student.ColorWeight
	tasteWeight := studentStatus.Student.TasteWeight
	satietyWeight := studentStatus.Student.SatietyWeight

	for len(planned) < 2 {
		var candidates []int
		var scores []float64

		for i, dish := range menu {
			if dish.Course != Starter || portions[i] <= 0 {
				continue
			}

			score := StarterScore(
				dish, colorWeight, tasteWeight,
				satietyWeight, portions[i],
			)

			// Reduce repeated selection score by 40%.
			if len(planned) > 0 && dish.ID == planned[0] {
				score *= 0.6
			}

			candidates = append(candidates, i)
			scores = append(scores, score)
		}

		if len(candidates) == 0 {
			break
		}

		selected := randomness.SoftmaxChoice(
			scores,
			temperature,
			studentStatus.randomGenerator,
		)
		if selected < 0 {
			break
		}

		index := candidates[selected]
		planned = append(planned, menu[index].ID)
		portions[index]--
	}

	return planned
}

package agents

import (
	"math/rand/v2"
	"time"
)

const saucePreferenceProbability = 0.50 // how likely are students want sauce?

// StudentGenerator creates one student's attributes at the moment of arrival.
type StudentGenerator struct {
	basePatience time.Duration
}

func NewStudentGenerator(basePatience time.Duration) *StudentGenerator {
	return &StudentGenerator{
		basePatience: basePatience,
	}
}

func (g *StudentGenerator) Generate(id int, arrivalTime time.Duration) StudentData {
	patienceScale := 0.4 + 1.2*rand.Float64()

	return StudentData{
		ID:          id,
		ArrivalTime: arrivalTime,

		Patience: time.Duration(float64(g.basePatience) * patienceScale),

		Vegetarian:    rand.Float64() < 0.35,
		ColorWeight:   rand.Float64(),
		TasteWeight:   rand.Float64(),
		SatietyWeight: rand.Float64(),
		WantsSauce:    rand.Float64() < saucePreferenceProbability,
	}
}

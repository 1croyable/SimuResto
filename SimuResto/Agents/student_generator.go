package agents

import (
	"math/rand"
	"time"
)

const saucePreferenceProbability = 0.50 // how likely are students want sauce?

// StudentGenerator creates one student's attributes at the moment of arrival.
type StudentGenerator struct {
	rng          *rand.Rand
	basePatience time.Duration
}

func NewStudentGenerator(seed int64, basePatience time.Duration) *StudentGenerator {
	return &StudentGenerator{
		rng:          rand.New(rand.NewSource(seed ^ 0x4e157cd2)),
		basePatience: basePatience,
	}
}

func (g *StudentGenerator) Generate(id int, arrivalTime time.Duration) StudentData {
	patienceScale := 0.4 + 1.2*g.rng.Float64()

	return StudentData{
		ID:          id,
		ArrivalTime: arrivalTime,

		Patience: time.Duration(float64(g.basePatience) * patienceScale),

		Vegetarian:    g.rng.Float64() < 0.35,
		ColorWeight:   g.rng.Float64(),
		TasteWeight:   g.rng.Float64(),
		SatietyWeight: g.rng.Float64(),
		WantsSauce:    g.rng.Float64() < saucePreferenceProbability,
	}
}

package randomness

import (
	"math"
	"time"
)

// model P(give up waiting | StudentState, QueueState)
func LeaveProbability(basePatience, waiting time.Duration, peopleAhead int) float64 {
	if waiting <= 0 {
		return 0
	}

	ahead := float64(max(0, peopleAhead))

	// Increasing impatience as waiting approaches personal patience.
	z := (waiting.Seconds() + ahead -
		basePatience.Seconds()) / 30.0

	impatience := 1 / (1 + math.Exp(-z))

	// increasing with the number of people ahead.
	baselineHazard := 0.0001 + 0.00005*ahead

	// Combined probability over one simulated second. F(t) = 1 - e^{-lambda t}
	return 1 - (1-impatience)*math.Exp(-baselineHazard)
}

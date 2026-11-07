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
	z := (waiting.Seconds() + 1.0*float64(peopleAhead) - basePatience.Seconds()) / 30.0
	return 1 / (1 + math.Exp(-z))
}

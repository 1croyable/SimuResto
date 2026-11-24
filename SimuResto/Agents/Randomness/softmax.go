package randomness

import (
	"math"
	"math/rand/v2"
)

// temperature = 0: always choose the highest score.
// temperature > 0: sample according to softmax probabilities.
// A lower temperature favors higher-scoring items.
func SoftmaxChoice(scores []float64, temperature float64) int {
	if len(scores) == 0 {
		return -1
	}

	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}

	if temperature <= 0 {
		return best
	}

	// softmax
	weights := make([]float64, len(scores))
	total := 0.0

	for i, score := range scores {
		weights[i] = math.Exp((score - scores[best]) / temperature)
		total += weights[i]
	}

	// Sample according to the probability distribution.
	draw := rand.Float64() * total

	for i, weight := range weights {
		draw -= weight
		if draw < 0 {
			return i
		}
	}

	return len(scores) - 1
}

package randomness

import (
	food "simuresto/SimuResto/Shared/Food"
)

// model P(choose dish | student preferences, available portions)
func StarterScore(dish food.Dish, colorWeight, tasteWeight, satietyWeight float64, portions int) float64 {
	if portions <= 0 {
		return 0
	}

	totalWeight := colorWeight + tasteWeight + satietyWeight
	if totalWeight <= 0 {
		return 0
	}

	preference := (colorWeight*dish.ColorScore + tasteWeight*dish.TasteScore + satietyWeight*dish.SatietyScore) / totalWeight

	stockFactor := float64(portions) / float64(portions+1)

	return preference * stockFactor
}

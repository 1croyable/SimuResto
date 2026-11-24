package agents

import randomness "simuresto/SimuResto/Agents/Randomness"

// model P(dish | student preferences, available portions)
func StarterScore(dish Dish, colorWeight, tasteWeight, satietyWeight float64, portions int) float64 {
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

func (studentStatus *StudentState) chooseDish(dishes []Dish, temperature float64) string {
	if len(dishes) == 0 {
		return ""
	}

	scores := make([]float64, len(dishes))
	student := studentStatus.Student

	for i, dish := range dishes {
		scores[i] = StarterScore(
			dish,
			student.ColorWeight,
			student.TasteWeight,
			student.SatietyWeight,
			dish.Portions,
		)
	}

	selected := randomness.SoftmaxChoice(scores, temperature)

	if selected < 0 {
		return ""
	}

	return dishes[selected].ID
}

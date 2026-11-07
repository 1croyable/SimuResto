package agents

// Dish is food data, rather than an agent that makes decisions.
type Dish struct {
	ID                string
	Name              string
	Course            Course
	Vegetarian        bool
	Portions          int     // Initial number of available portions.
	SatisfactionScore float32 // Dish rating between 0 and 100.
}

// Meal contains the dishes collected by a student.
type Meal struct {
	Starters []Dish
	Main     *Dish // nil until a main dish has been collected.
	Desserts []Dish
}

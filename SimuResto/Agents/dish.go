package agents

type Course string

const (
	Starter Course = "starter"
	Main    Course = "main"
)

type Dish struct {
	ID         string
	Name       string
	Course     Course
	Vegetarian bool
	Portions   int // Number of available portions.

	ColorScore   float64 // 0–1
	TasteScore   float64 // 0–1
	SatietyScore float64 // 0–1

	StarterCounter int // 1 or 2 for starters; 0 for main courses
}

type Meal struct {
	Starters []Dish
	Main     *Dish
}

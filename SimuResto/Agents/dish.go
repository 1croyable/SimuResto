package agents

type Dish struct {
	ID         string
	Name       string
	Course     Course
	Vegetarian bool
	Portions   int // Number of available portions.
}

type Meal struct {
	Starters []Dish
	Main     *Dish
	Desserts []Dish
}

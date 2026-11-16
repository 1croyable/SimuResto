package food

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
}

type Meal struct {
	Starters []Dish
	Main     *Dish
}

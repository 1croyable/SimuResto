package agents

import "time"

type Point struct {
	X, Y float64
}

type StudentStatus string

const (
	Walking StudentStatus = "walking"
	Serving StudentStatus = "receiving a dish"
	Served  StudentStatus = "has a dish"
)

// StudentState holds properties that change during a simulation.
type StudentState struct {
	Student      Student
	Position     Point
	NextWaypoint int
	Status       StudentStatus
	Meal         Meal
}

type StaffState struct {
	Staff            Staff
	Position         Point
	RemainingService time.Duration
}

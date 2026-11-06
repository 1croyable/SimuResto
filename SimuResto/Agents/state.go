package agents

import "time"

type Point struct {
	X, Y float64
}

type StudentStatus string

const (
	Walking          StudentStatus = "walking"
	Serving          StudentStatus = "receiving a dish"
	WalkingToCashier StudentStatus = "walking to checkout"
	Paying           StudentStatus = "paying"
	WalkingToDining  StudentStatus = "walking to dining"
	Dining           StudentStatus = "dining"
	Finished         StudentStatus = "finished"
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

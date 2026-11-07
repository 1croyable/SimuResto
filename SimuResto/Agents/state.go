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
	WalkingToTrash   StudentStatus = "walking to waste station"
	WalkingToExit    StudentStatus = "walking to exit"
	Finished         StudentStatus = "finished"
)

// StudentState holds properties that change during a simulation.
type StudentState struct {
	Student         StudentData
	Position        Point
	NextWaypoint    int
	Status          StudentStatus
	Meal            Meal
	RemainingDining time.Duration
}

type StaffState struct {
	Staff            Staff
	Position         Point
	RemainingService time.Duration
}

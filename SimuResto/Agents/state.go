package agents

import (
	"math/rand"
	"time"
)

type Point struct {
	X, Y float64
}

// StudentState holds properties that change during a simulation.
type StudentState struct {
	Student           StudentData
	Position          Point
	NextWaypoint      int
	Status            StudentStatus
	Meal              Meal
	RemainingDining   time.Duration
	CounterReleasedAt time.Duration // Simulated time when the student last left a service counter.

	// PositionAlongRoute is measured from the U-shaped route's start in the direction of travel.
	// Negative values place the student before the route's start.
	PositionAlongRoute float64
	QueueWait          time.Duration
	OrderedDishID      string
	randomGenerator    *rand.Rand
}

type StaffState struct {
	StudentID       int
	Staff           StaffData
	Position        Point
	ServiceTimeLeft time.Duration
}

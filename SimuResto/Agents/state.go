package agents

import "time"

type Point struct {
	X, Y float64
}

// StudentState holds properties that change during a simulation.
type StudentState struct {
	Student  StudentData
	Position Point

	NextWaypoint int
	Target       StudentTarget

	Status            StudentStatus
	Meal              Meal
	DiningTime        time.Duration
	RemainingDining   time.Duration
	SeatID            int // Zero until a seat is assigned.
	DiningNode        int // Node at the end of the current approach or waiting segment.
	DiningRoute       []Point
	CounterReleasedAt time.Duration // Simulated time when the student last left a service counter.

	// PositionAlongRoute is measured from the U-shaped route's start in the direction of travel.
	// Negative values place the student before the route's start.
	PositionAlongRoute float64
	QueueWait          time.Duration
	OrderedDishID      string

	RemainingStarterPickup time.Duration // collecting starters may take several simulation updates

	PlannedStarterIDs []string

	CashierIndex int // 0 lower cahier, 1 upper

	RetrySauce           bool
	RemainingSaucePickup time.Duration

	AbandonRoute []Point
}

type StaffState struct {
	StudentID       int
	Staff           StaffData
	Position        Point
	ServiceTimeLeft time.Duration
}

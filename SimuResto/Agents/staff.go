package agents

import "time"

type StaffData struct {
	ID          int
	Role        Role
	ServiceTime time.Duration // Time needed to serve one student.
}

type RestockOption struct {
	CounterIndex int
	DishID       string
}

type RestockObservation struct {
	Active        bool
	CounterIndex  int
	DishID        string
	RemainingTime time.Duration
	Duration      time.Duration
	Options       []RestockOption
}

type StaffObservation struct {
	NextStudentID int // First waiting student; 0 if none.
	QueueLength   int
	Elapsed       time.Duration
	Open          bool
	Restock       *RestockObservation // when Restock != nil, we need to refill
}

type StaffDecision struct {
	AcceptedStudentID  int
	CompletedStudentID int
	BusyTime           time.Duration
	RemainingTime      time.Duration

	RestockDishID    string
	RestockCounter   int
	RestockCompleted bool
}

func (staff *StaffState) AdvanceService(dt time.Duration) time.Duration {
	busyTime := min(dt, staff.ServiceTimeLeft)
	staff.ServiceTimeLeft -= busyTime
	return busyTime
}

func (staff *StaffState) Decide(obs StaffObservation) StaffDecision {
	if obs.Restock != nil {
		return staff.decideRestock(*obs.Restock, obs.Elapsed)
	}
	decision := StaffDecision{
		RemainingTime: staff.ServiceTimeLeft,
	}

	studentID := staff.StudentID

	// Accept a new student only when idle and open.
	if studentID == 0 {
		if !obs.Open || obs.NextStudentID == 0 {
			return decision
		}

		studentID = obs.NextStudentID
		decision.AcceptedStudentID = studentID
		decision.RemainingTime = staff.Staff.ServiceTime
	}

	elapsed := max(obs.Elapsed, 0)
	decision.BusyTime = min(elapsed, decision.RemainingTime)
	decision.RemainingTime -= decision.BusyTime

	if decision.RemainingTime <= 0 {
		decision.RemainingTime = 0
		decision.CompletedStudentID = studentID
	}

	return decision
}

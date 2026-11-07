package agents

import "time"

type StaffData struct {
	ID          int
	Role        Role
	ServiceTime time.Duration // Time needed to serve one student.
}

func (staff *StaffState) AdvanceService(dt time.Duration) time.Duration {
	busyTime := min(dt, staff.ServiceTimeLeft)
	staff.ServiceTimeLeft -= busyTime
	return busyTime
}

package engine

import (
	"math/rand/v2"
	"time"

	agents "simuresto/SimuResto/Agents"
)

func randomServiceTime(minimum, maximum time.Duration) time.Duration {
	return minimum + time.Duration(rand.Int64N(int64(maximum-minimum)+1))
}

func startCounterService(staff *agents.StaffState, studentID int) {
	avg := staff.Staff.ServiceTime
	variation := avg / 4
	staff.StudentID = studentID
	staff.ServiceDuration = randomServiceTime(avg-variation, avg+variation)
	staff.ServiceTimeLeft = staff.ServiceDuration
}

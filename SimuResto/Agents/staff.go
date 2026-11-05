package agents

import "time"

type Staff struct {
	ID          int
	Role        Role
	ServiceTime time.Duration // Time the staff still needs to finish serving
}

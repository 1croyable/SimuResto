package agents

import "time"

type Staff struct {
	ID          int
	Role        Role
	ServiceTime time.Duration
}

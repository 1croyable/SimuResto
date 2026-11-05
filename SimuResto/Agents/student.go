package agents

import "time"

type Student struct {
	ID             int
	Preference     Preference
	ArrivalTime    time.Duration
	StartersWanted int
	DessertsWanted int
}

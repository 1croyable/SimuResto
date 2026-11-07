package environment

import "time"

type Settings struct {
	StudentCount    int
	ArrivalInterval time.Duration
	Seed            int64
	BasePatience    time.Duration
	QueueSpacing    float64
	FramesPerSecond int
	TicksPerSecond  int
	WalkingSpeed    float64 // map units in one simulated second
	ServiceTime     time.Duration
	PaymentTime     time.Duration
	DiningTime      time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		StudentCount:    100,
		ArrivalInterval: 0,
		Seed:            42,
		BasePatience:    10 * time.Minute,
		QueueSpacing:    16,
		FramesPerSecond: 60,
		TicksPerSecond:  60,
		WalkingSpeed:    180,
		ServiceTime:     2 * time.Second,
		PaymentTime:     4 * time.Second,
		DiningTime:      5 * time.Second,
	}
}

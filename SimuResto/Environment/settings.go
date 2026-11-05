package environment

import "time"

type Settings struct {
	FramesPerSecond int
	TicksPerSecond  int
	WalkingSpeed    float64 // map units in one simulated second
	ServiceTime     time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		FramesPerSecond: 60,
		TicksPerSecond:  60,
		WalkingSpeed:    180,
		ServiceTime:     2 * time.Second,
	}
}

package environment

import (
	"time"
)

// Default configuration values and simulation constants.
const (
	DefaultCashiers = 2
	MaxCashiers     = 2
)

type Settings struct {
	StartTime time.Duration // Simulation start time.

	// Students
	StudentCount int
	Seed         int64
	BasePatience time.Duration

	// Movement and queues
	QueueSpacing float64
	WalkingSpeed float64 // Map units per simulated second.

	// Simulation timing and rendering
	FramesPerSecond int
	TicksPerSecond  int

	// Staff and service
	Cashiers        int
	ServiceTime     time.Duration
	RestockDuration time.Duration

	PaymentTime time.Duration
	DiningTime  time.Duration

	ChoiceTemperature float64 // food selection

	Lunch       ArrivalPeriod
	Dinner      ArrivalPeriod
	DinnerShare float64 // Fraction of StudentCount assigned to dinner.
}

type ArrivalPeriod struct {
	Open, Close, Peak     time.Duration
	EarlyLead             time.Duration
	SigmaLeft, SigmaRight time.Duration
	EarlyRatio            float64
	UniformRatio          float64
	StopBeforeClose       time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		StartTime: 11 * time.Hour,

		StudentCount: 100,
		Seed:         42,
		BasePatience: 10 * time.Minute,

		QueueSpacing: 16,
		WalkingSpeed: 180,

		FramesPerSecond: 60,
		TicksPerSecond:  60,

		Cashiers:        DefaultCashiers,
		ServiceTime:     2 * time.Second,
		RestockDuration: 10 * time.Second,

		PaymentTime: 4 * time.Second,
		DiningTime:  5 * time.Second,

		ChoiceTemperature: 0.3,

		Lunch: ArrivalPeriod{
			Open:            11*time.Hour + 30*time.Minute, // 11:30
			Close:           13*time.Hour + 45*time.Minute, // 13:45
			Peak:            12*time.Hour + 10*time.Minute, // 12:10
			EarlyLead:       15 * time.Minute,
			SigmaLeft:       20 * time.Minute,
			SigmaRight:      35 * time.Minute,
			EarlyRatio:      0.02,
			UniformRatio:    0.20,
			StopBeforeClose: 10 * time.Minute,
		},

		Dinner: ArrivalPeriod{
			Open:            18*time.Hour + 15*time.Minute, // 18:15
			Close:           20*time.Hour + 30*time.Minute, // 20:30
			Peak:            18*time.Hour + 30*time.Minute, // 18:30
			EarlyLead:       10 * time.Minute,
			SigmaLeft:       20 * time.Minute,
			SigmaRight:      30 * time.Minute,
			EarlyRatio:      0.02,
			UniformRatio:    0.20,
			StopBeforeClose: 10 * time.Minute,
		},

		DinnerShare: 0.35,
	}
}

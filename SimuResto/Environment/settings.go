package environment

import (
	"time"
)

// Default configuration values and simulation constants.
const (
	DefaultStudentCount  = 1600
	OpeningTime          = 11*time.Hour + 30*time.Minute
	ClosingTime          = 13*time.Hour + 45*time.Minute
	ArrivalPeakTime      = 12*time.Hour + 15*time.Minute
	ArrivalSigma         = 20 * time.Minute
	OpeningStudentRatio  = 0.02
	UniformArrivalRatio  = 0.20
	DefaultTick          = 250 * time.Millisecond
	ServiceDuration      = ClosingTime - OpeningTime
	DefaultPlaybackSpeed = 10.0
	DefaultCashiers      = 2
	MaxCashiers          = 2

	OutdoorCapacity     = 80
	IndoorCapacity      = 20
	SelfServiceCapacity = 6
	MainServiceCapacity = 3
	Servers             = 2

	// Students may abandon the outdoor queue due to insufficient
	// patience or satisfaction only when occupancy strictly
	// exceeds this percentage of its capacity.
	OutdoorAbandonmentThresholdPercent = 30

	// Each service duration is sampled uniformly within
	// the corresponding range when service begins.
	MinSelfServiceTime = 8 * time.Second
	MaxSelfServiceTime = 14 * time.Second
	MinMainServiceTime = 6 * time.Second
	MaxMainServiceTime = 10 * time.Second
	MinPaymentTime     = 8 * time.Second
	MaxPaymentTime     = 14 * time.Second

	MainWeight                float32 = 2
	StarterWeight             float32 = 1
	IndoorLossPerSecond       float32 = 0.10
	OutdoorLossPerSecond      float32 = 0.35
	MissingSidePenalty        float32 = 5
	PreferenceMismatchPenalty float32 = 30
	MaxMealSatisfactionChange float32 = 50

	StarterPortions = 300
)

// Settings contains the configurable parameters for a simulation run.
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
	Tick            time.Duration

	// Staff and service
	Cashiers    int
	ServiceTime time.Duration
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
		Tick:            DefaultTick,

		Cashiers:    DefaultCashiers,
		ServiceTime: 2 * time.Second,
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

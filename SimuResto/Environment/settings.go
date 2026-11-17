package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
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
	MinArrivalInterval   = time.Second
	DefaultTick          = 250 * time.Millisecond
	ServiceDuration      = ClosingTime - OpeningTime
	DefaultPlaybackSpeed = 4.0
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

	// CROUS meal composition: one main course and at most
	// two additional items combining starters and desserts.
	MaxSideItems = 2

	MainWeight                float32 = 2
	StarterWeight             float32 = 1
	DessertWeight             float32 = 1
	IndoorLossPerSecond       float32 = 0.10
	OutdoorLossPerSecond      float32 = 0.35
	MissingSidePenalty        float32 = 5
	PreferenceMismatchPenalty float32 = 30
	MaxMealSatisfactionChange float32 = 50

	StarterPortions = 300
	DessertPortions = 350
)

// Settings contains the configurable parameters for a simulation run.
type Settings struct {
	// Students
	StudentCount    int
	ArrivalInterval time.Duration
	Seed            int64
	BasePatience    time.Duration

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
}

func DefaultSettings() Settings {
	return Settings{
		StudentCount:    100,
		ArrivalInterval: 0,
		Seed:            42,
		BasePatience:    10 * time.Minute,

		QueueSpacing: 16,
		WalkingSpeed: 180,

		FramesPerSecond: 60,
		TicksPerSecond:  60,
		Tick:            DefaultTick,

		Cashiers:    DefaultCashiers,
		ServiceTime: 2 * time.Second,
		PaymentTime: 4 * time.Second,
		DiningTime:  5 * time.Second,
	}
}

// DefaultStudentGenerator uses a fixed arrival distribution
// and schedules student arrivals according to the simulation tick.
func DefaultStudentGenerator(tick time.Duration) agents.StudentGenerator {
	return agents.StudentGenerator{
		Tick:            tick,
		ArrivalDuration: ServiceDuration,
		ArrivalPeak:     ArrivalPeakTime - OpeningTime,
		ArrivalSigma:    ArrivalSigma,
		OpeningRatio:    OpeningStudentRatio,
		UniformRatio:    UniformArrivalRatio,
	}
}

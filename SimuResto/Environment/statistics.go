package environment

import "time"

type CounterStatistics struct {
	StudentsServed   int
	DishesCollected  int
	TotalWaitTime    time.Duration
	TotalServiceTime time.Duration
	MaxWaitTime      time.Duration
}

// Global
type Statistics struct {
	StarterCounters [2]CounterStatistics
	MainCounter     CounterStatistics
	CashierCounter  CounterStatistics

	MaxQueueLength int

	CookRestockTime time.Duration
	RestockBatches  int
}

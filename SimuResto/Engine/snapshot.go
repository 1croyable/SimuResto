package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type StudentSnapshot struct {
	Student                agents.StudentData
	Position               agents.Point
	Status                 agents.StudentStatus
	Target                 agents.StudentTarget
	NextWaypoint           int
	PositionAlongRoute     float64
	QueueWait              time.Duration
	OrderedDishID          string
	PlannedStarterIDs      []string
	RemainingStarterPickup time.Duration
	RemainingDining        time.Duration
	CounterReleasedAt      time.Duration
	Meal                   agents.Meal
	IsLeaving              bool
}

type WorkerSnapshot struct {
	Staff           agents.StaffData
	Position        agents.Point
	StudentID       int
	ServiceTimeLeft time.Duration
	Served          int
	BusyTime        time.Duration
	Utilization     float64 // From completed service, in percent.
}

type StockSnapshot struct {
	Dish      agents.Dish // Dish.Portions is the currently available stock.
	Reserved  int         // Starters chosen but not yet collected.
	Collected int         // Portions in student meals.
}

type QueueSnapshot struct {
	StudentIDs []int
	Capacity   int
}

type ZoneSnapshot struct {
	Target     agents.StudentTarget
	StudentIDs []int
}

type DiningAreaSnapshot struct {
	Name              string
	Region            environment.Region
	Route             []agents.Point
	CleanupRoute      []agents.Point
	UpperCashierRoute []agents.Point
}

type LayoutSnapshot struct {
	Roads          []environment.Region
	StarterRegions [2]environment.Region
	MainRegion     environment.Region
	DiningAreas    [3]DiningAreaSnapshot
	WasteRegion    environment.Region
	Route          []agents.Point
	Starter2Route  []agents.Point
	MainRoute      []agents.Point

	Cashier1Region environment.Region
	Checkout1Route []agents.Point
	ExitRoute      []agents.Point

	Cashier2Region    environment.Region
	Checkout2Route    []agents.Point
	Cashier2ExitRoute []agents.Point
}

type Metrics struct {
	TotalStudents        int
	Scheduled            int
	Active               int
	Completed            int
	Abandoned            int
	Admitted             int // Students no longer in Scheduled status.
	QueueLength          int
	QueueCapacity        int
	MaxQueueLength       int
	MealsServed          int     // Completed payments, not students who finished dining.
	AbandonmentRate      float64 // Abandoned / Admitted.
	MealsPerMinute       float64
	AverageServiceWait   time.Duration // Average per completed service, not per student.
	AverageWaitByCounter map[string]time.Duration
	MaxWaitByCounter     map[string]time.Duration
	StaffUtilization     map[agents.Role]float64
}

type Report struct {
	Duration       time.Duration
	Settings       environment.Settings
	Students       []StudentSnapshot
	Staff          []WorkerSnapshot
	Stock          []StockSnapshot
	Statistics     environment.Statistics
	Metrics        Metrics
	PeakOccupancy  map[agents.StudentTarget]int // Maximum observed at Step boundaries.
	OperatingPhase environment.OperatingPhase
}

type Snapshot struct {
	Report
	Layout LayoutSnapshot
	Queue  QueueSnapshot
	Zones  []ZoneSnapshot
	Done   bool
}

func averageCounterWait(c environment.CounterStatistics) time.Duration {
	if c.StudentsServed == 0 {
		return 0
	}
	return c.TotalWaitTime / time.Duration(c.StudentsServed)
}
func staffUtilization(c environment.CounterStatistics, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return 100 * float64(c.TotalServiceTime) / float64(elapsed)
}

func buildSnapshot(world *environment.Environment, done bool, previous Snapshot) Snapshot {
	s := Snapshot{
		Done: done,
		Report: Report{
			Duration:       world.TimePassed,
			Settings:       world.Settings,
			Statistics:     world.Statistics,
			Students:       make([]StudentSnapshot, 0, len(world.StudentStates)),
			Stock:          make([]StockSnapshot, 0, len(world.Menu)),
			PeakOccupancy:  make(map[agents.StudentTarget]int),
			OperatingPhase: world.OperatingPhase,
		},
		Layout: LayoutSnapshot{
			Roads:             append([]environment.Region(nil), world.Roads...),
			StarterRegions:    world.StarterRegions,
			MainRegion:        world.MainRegion,
			Cashier1Region:    world.CashierRegion,
			WasteRegion:       world.WasteRegion,
			Route:             append([]agents.Point(nil), world.Route...),
			Starter2Route:     append([]agents.Point(nil), world.Starter2Route...),
			MainRoute:         append([]agents.Point(nil), world.MainRoute...),
			Checkout1Route:    append([]agents.Point(nil), world.CheckoutRoute...),
			ExitRoute:         append([]agents.Point(nil), world.ExitRoute...),
			Cashier2Region:    world.Cashier2Region,
			Checkout2Route:    append([]agents.Point(nil), world.Checkout2Route...),
			Cashier2ExitRoute: append([]agents.Point(nil), world.Cashier2ExitRoute...),
		},
	}
	for i, area := range world.DiningAreas {
		s.Layout.DiningAreas[i] = DiningAreaSnapshot{
			Name: area.Name, Region: area.Region,
			Route:             append([]agents.Point(nil), area.Route...),
			CleanupRoute:      append([]agents.Point(nil), area.CleanupRoute...),
			UpperCashierRoute: append([]agents.Point(nil), area.UpperCashierRoute...),
		}
	}
	if world.Queue != nil {
		s.Queue.StudentIDs = world.Queue.IDs()
		s.Queue.Capacity = world.Queue.Capacity()
	}
	s.Metrics = Metrics{
		TotalStudents:        len(world.StudentStates),
		QueueLength:          len(s.Queue.StudentIDs),
		QueueCapacity:        s.Queue.Capacity,
		MaxQueueLength:       world.Statistics.MaxQueueLength,
		MealsServed:          world.Statistics.CashierCounter.StudentsServed,
		AverageWaitByCounter: make(map[string]time.Duration),
		MaxWaitByCounter:     make(map[string]time.Duration),
		StaffUtilization:     make(map[agents.Role]float64),
	}
	reserved := make(map[string]int)
	collected := make(map[string]int)
	for _, student := range world.StudentStates {
		item := StudentSnapshot{
			Student: student.Student, Position: student.Position,
			Status: student.Status, Target: student.Target,
			NextWaypoint:           student.NextWaypoint,
			PositionAlongRoute:     student.PositionAlongRoute,
			QueueWait:              student.QueueWait,
			OrderedDishID:          student.OrderedDishID,
			PlannedStarterIDs:      append([]string(nil), student.PlannedStarterIDs...),
			RemainingStarterPickup: student.RemainingStarterPickup,
			RemainingDining:        student.RemainingDining,
			CounterReleasedAt:      student.CounterReleasedAt,
			Meal:                   agents.Meal{Starters: append([]agents.Dish(nil), student.Meal.Starters...)},
			IsLeaving:              len(student.AbandonRoute) > 0,
		}
		if student.Meal.Main != nil {
			main := *student.Meal.Main
			item.Meal.Main = &main
			collected[main.ID]++
		}
		for _, dish := range student.Meal.Starters {
			collected[dish.ID]++
		}
		for _, id := range student.PlannedStarterIDs {
			reserved[id]++
		}
		s.Students = append(s.Students, item)
		switch student.Status {
		case agents.Scheduled:
			s.Metrics.Scheduled++
		case agents.Finished:
			s.Metrics.Completed++
			s.Metrics.Admitted++
		case agents.Abandoned:
			s.Metrics.Abandoned++
			s.Metrics.Admitted++
		default:
			s.Metrics.Active++
			s.Metrics.Admitted++
		}
	}
	for _, dish := range world.Menu {
		s.Stock = append(s.Stock, StockSnapshot{Dish: dish, Reserved: reserved[dish.ID], Collected: collected[dish.ID]})
	}
	zoneIndex := make(map[agents.StudentTarget]int)
	for _, id := range s.Queue.StudentIDs {
		student, exists := world.StudentsByID[id]
		if !exists {
			continue
		}
		target := student.Target
		i, exists := zoneIndex[target]
		if !exists {
			i = len(s.Zones)
			zoneIndex[target] = i
			s.Zones = append(s.Zones, ZoneSnapshot{Target: target})
		}
		s.Zones[i].StudentIDs = append(s.Zones[i].StudentIDs, id)
	}
	for k, v := range previous.PeakOccupancy {
		s.PeakOccupancy[k] = v
	}
	for _, z := range s.Zones {
		if len(z.StudentIDs) > s.PeakOccupancy[z.Target] {
			s.PeakOccupancy[z.Target] = len(z.StudentIDs)
		}
	}

	counterNames := []string{"Starter 1", "Starter 2", "Main", "Cashier"}
	counters := []environment.CounterStatistics{
		world.Statistics.StarterCounters[0], world.Statistics.StarterCounters[1],
		world.Statistics.MainCounter, world.Statistics.CashierCounter,
	}
	var totalWait time.Duration
	served := 0
	for i, c := range counters {
		s.Metrics.AverageWaitByCounter[counterNames[i]] = averageCounterWait(c)
		s.Metrics.MaxWaitByCounter[counterNames[i]] = c.MaxWaitTime
		totalWait += c.TotalWaitTime
		served += c.StudentsServed
	}
	if served > 0 {
		s.Metrics.AverageServiceWait = totalWait / time.Duration(served)
	}
	if s.Metrics.Admitted > 0 {
		s.Metrics.AbandonmentRate = float64(s.Metrics.Abandoned) / float64(s.Metrics.Admitted)
	}
	if world.TimePassed > 0 {
		s.Metrics.MealsPerMinute = float64(s.Metrics.MealsServed) / world.TimePassed.Minutes()
	}
	s.Metrics.StaffUtilization[agents.Server] = staffUtilization(world.Statistics.MainCounter, world.TimePassed)
	s.Metrics.StaffUtilization[agents.Cashier] = staffUtilization(world.Statistics.CashierCounter, world.TimePassed)
	// Cook
	s.Staff = []WorkerSnapshot{
		{
			Staff:           world.Cook.Staff,
			Position:        world.Cook.Position,
			StudentID:       world.Cook.StudentID,
			ServiceTimeLeft: world.Cook.ServiceTimeLeft,
			Served:          world.Statistics.MainCounter.StudentsServed,
			BusyTime:        world.Statistics.MainCounter.TotalServiceTime,
			Utilization:     s.Metrics.StaffUtilization[agents.Server],
		},
	}

	// Cashiers
	for i := range world.Cashiers {
		cashier := &world.Cashiers[i]

		// Hide inactive cashiers unless they are finishing a service.
		if i >= world.Settings.Cashiers && cashier.StudentID == 0 {
			continue
		}

		s.Staff = append(s.Staff, WorkerSnapshot{
			Staff:           cashier.Staff,
			Position:        cashier.Position,
			StudentID:       cashier.StudentID,
			ServiceTimeLeft: cashier.ServiceTimeLeft,
		})
	}
	return s
}

func cloneSnapshot(source Snapshot) Snapshot {
	s := source
	s.Students = append([]StudentSnapshot(nil), source.Students...)
	for i := range s.Students {
		s.Students[i].PlannedStarterIDs = append([]string(nil), source.Students[i].PlannedStarterIDs...)
		s.Students[i].Meal.Starters = append([]agents.Dish(nil), source.Students[i].Meal.Starters...)
		if source.Students[i].Meal.Main != nil {
			main := *source.Students[i].Meal.Main
			s.Students[i].Meal.Main = &main
		}
	}
	s.Staff = append([]WorkerSnapshot(nil), source.Staff...)
	s.Stock = append([]StockSnapshot(nil), source.Stock...)
	s.Queue.StudentIDs = append([]int(nil), source.Queue.StudentIDs...)
	s.Zones = append([]ZoneSnapshot(nil), source.Zones...)
	for i := range s.Zones {
		s.Zones[i].StudentIDs = append([]int(nil), source.Zones[i].StudentIDs...)
	}
	s.Layout.Roads = append([]environment.Region(nil), source.Layout.Roads...)
	s.Layout.Route = append([]agents.Point(nil), source.Layout.Route...)
	s.Layout.Starter2Route = append([]agents.Point(nil), source.Layout.Starter2Route...)
	s.Layout.MainRoute = append([]agents.Point(nil), source.Layout.MainRoute...)
	s.Layout.Checkout1Route = append([]agents.Point(nil), source.Layout.Checkout1Route...)
	s.Layout.ExitRoute = append([]agents.Point(nil), source.Layout.ExitRoute...)
	for i := range s.Layout.DiningAreas {
		s.Layout.DiningAreas[i].Route = append([]agents.Point(nil), source.Layout.DiningAreas[i].Route...)
		s.Layout.DiningAreas[i].CleanupRoute = append([]agents.Point(nil), source.Layout.DiningAreas[i].CleanupRoute...)
	}
	s.Layout.Checkout2Route = append([]agents.Point(nil), source.Layout.Checkout2Route...)
	s.Layout.Cashier2ExitRoute = append([]agents.Point(nil), source.Layout.Cashier2ExitRoute...)

	for i := range s.Layout.DiningAreas {
		s.Layout.DiningAreas[i].UpperCashierRoute = append(
			[]agents.Point(nil),
			source.Layout.DiningAreas[i].UpperCashierRoute...,
		)
	}

	s.PeakOccupancy = make(map[agents.StudentTarget]int, len(source.PeakOccupancy))
	for k, v := range source.PeakOccupancy {
		s.PeakOccupancy[k] = v
	}

	s.Metrics.AverageWaitByCounter = make(map[string]time.Duration, len(source.Metrics.AverageWaitByCounter))
	for k, v := range source.Metrics.AverageWaitByCounter {
		s.Metrics.AverageWaitByCounter[k] = v
	}
	s.Metrics.MaxWaitByCounter = make(map[string]time.Duration, len(source.Metrics.MaxWaitByCounter))
	for k, v := range source.Metrics.MaxWaitByCounter {
		s.Metrics.MaxWaitByCounter[k] = v
	}
	s.Metrics.StaffUtilization = make(map[agents.Role]float64, len(source.Metrics.StaffUtilization))
	for k, v := range source.Metrics.StaffUtilization {
		s.Metrics.StaffUtilization[k] = v
	}
	return s
}

func (g *Game) Snapshot() Snapshot {
	return cloneSnapshot(g.snapshot)
}

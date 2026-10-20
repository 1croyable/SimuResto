package simulation

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func testStudents(count int) []Student {
	students := make([]Student, count)
	for i := range students {
		students[i] = Student{
			ID: i + 1, Preference: Vegetarian, Patience: time.Hour,
			Satisfaction: 80, StartersWanted: 1, DessertsWanted: 1,
		}
	}
	return students
}

func mustRun(t *testing.T, config Config, students []Student) Report {
	t.Helper()
	report, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	checkInvariants(t, config, report)
	return report
}

// Replay the public event log independently of the engine to check capacities,
// FIFO transfers, exclusive occupancy, stock conservation, and terminal states.
func checkInvariants(t *testing.T, config Config, report Report) {
	t.Helper()
	capacities := map[Area]int{
		Outside: config.Capacities.Outside, Inside: config.Capacities.Inside,
		SelfService: config.Capacities.SelfService, MainService: config.Capacities.MainService,
		Payment: config.Capacities.Payment,
	}
	queues := make(map[Area][]int)
	locations := make(map[int]Area)
	peaks := make(map[Area]int)
	var previousTime time.Duration
	remove := func(event Event, fifo bool) {
		t.Helper()
		if locations[event.StudentID] != event.From {
			t.Fatalf("event removes student from the wrong area: %+v", event)
		}
		queue := queues[event.From]
		found := -1
		for i, id := range queue {
			if id == event.StudentID {
				found = i
				break
			}
		}
		if found < 0 || (fifo && found != 0) {
			t.Fatalf("invalid FIFO removal: event=%+v queue=%v", event, queue)
		}
		queues[event.From] = append(queue[:found], queue[found+1:]...)
		delete(locations, event.StudentID)
	}
	for _, event := range report.Events {
		if event.Time < previousTime || event.Time > report.Duration {
			t.Fatalf("invalid event time: %+v", event)
		}
		previousTime = event.Time
		switch event.Signal {
		case Entered:
			if event.From != "" {
				remove(event, true)
			} else if _, exists := locations[event.StudentID]; exists {
				t.Fatalf("student %d occupies two areas", event.StudentID)
			}
			queues[event.To] = append(queues[event.To], event.StudentID)
			locations[event.StudentID] = event.To
			peaks[event.To] = max(peaks[event.To], len(queues[event.To]))
			if len(queues[event.To]) > capacities[event.To] {
				t.Fatalf("capacity exceeded in %s", event.To)
			}
		case Finished:
			remove(event, true)
		case LeftQueue:
			if event.From != Inside && event.From != Outside {
				t.Fatal("abandonment after tray collection")
			}
			remove(event, false)
		case Stopped:
			if event.From != "" {
				remove(event, false)
			}
		}
	}
	if len(locations) != 0 {
		t.Fatalf("unreleased places: %v", locations)
	}
	for area, capacity := range capacities {
		if report.PeakOccupancy[area] != peaks[area] || peaks[area] > capacity {
			t.Fatalf("invalid peak occupancy for %s", area)
		}
	}
	taken := make(map[string]int)
	for _, student := range report.Students {
		if student.Status == Scheduled || student.Status == Active || student.Area != "" {
			t.Fatalf("nonterminal student: %+v", student)
		}
		if student.Satisfaction < 0 || student.Satisfaction > 100 {
			t.Fatal("satisfaction out of bounds")
		}
		var wait time.Duration
		for _, duration := range student.Waiting {
			wait += duration
		}
		if wait != student.TotalWait || wait > student.ExitTime-student.ArrivalTime {
			t.Fatalf("invalid waiting times for student %d", student.Student.ID)
		}
		if len(student.Meal.Starters) > config.MaxStartersPerStudent ||
			len(student.Meal.Desserts) > config.MaxDessertsPerStudent ||
			len(student.Meal.Starters)+len(student.Meal.Desserts) > config.MaxSelfServiceItems {
			t.Fatal("self-service allowance exceeded")
		}
		dishes := append(append([]Dish(nil), student.Meal.Starters...), student.Meal.Desserts...)
		if student.Meal.Main != nil {
			dishes = append(dishes, *student.Meal.Main)
		}
		for _, dish := range dishes {
			taken[dish.ID]++
			if student.Student.Preference == Vegetarian && !dish.Vegetarian {
				t.Fatal("vegetarian student received meat")
			}
		}
		if student.Status == Completed {
			wantPath := []Area{Inside, SelfService, MainService, Payment}
			if len(student.Path) == 5 {
				wantPath = append([]Area{Outside}, wantPath...)
			}
			if !reflect.DeepEqual(student.Path, wantPath) {
				t.Fatalf("invalid completed route: %v", student.Path)
			}
		}
	}
	for _, stock := range report.Stock {
		if stock.Remaining < 0 || stock.Reserved != 0 || stock.Taken != taken[stock.Dish.ID] ||
			stock.Remaining+stock.Taken != stock.Dish.Portions {
			t.Fatalf("stock is not conserved: %+v", stock)
		}
	}
	for _, worker := range report.Staff {
		if worker.Utilization < 0 || worker.Utilization > 1 || worker.BusyTime > report.Duration {
			t.Fatalf("invalid worker utilization: %+v", worker)
		}
	}
	metrics := report.Metrics
	if metrics.Completed+metrics.Abandoned+metrics.Rejected+metrics.Cancelled != len(report.Students) {
		t.Fatal("outcome counts do not sum to the student population")
	}
}

func TestSingleStudentCompletesRoute(t *testing.T) {
	config := DefaultConfig()
	report := mustRun(t, config, testStudents(1))
	student := report.Students[0]
	if report.Duration != 16*time.Second || student.TotalWait != 0 || student.Status != Completed {
		t.Fatalf("unexpected single-student timing: %+v", student)
	}
	if student.Meal.Main == nil || !student.Meal.Main.Vegetarian || student.Satisfaction != 89 {
		t.Fatalf("unexpected meal or satisfaction: %+v", student)
	}
	var signals []Signal
	for _, event := range report.Events {
		if event.Signal == OrderPlaced || event.Signal == MainServed || event.Signal == PaymentStarted || event.Signal == Paid {
			signals = append(signals, event.Signal)
		}
	}
	if !reflect.DeepEqual(signals, []Signal{OrderPlaced, MainServed, PaymentStarted, Paid}) {
		t.Fatalf("invalid service handshake: %v", signals)
	}
	if report.Staff[0].BusyTime != 8*time.Second || report.Staff[2].BusyTime != 4*time.Second {
		t.Fatal("incorrect worker busy time")
	}
}

func TestCrowdIsBoundedAndFIFO(t *testing.T) {
	config := DefaultConfig()
	config.Menu[1].Portions = 100
	report := mustRun(t, config, testStudents(120))
	// At time zero: six collecting sides, twenty inside, fifty outside.
	if report.Metrics.Completed != 76 || report.Metrics.Rejected != 44 || report.Metrics.Abandoned != 0 {
		t.Fatalf("unexpected crowd outcomes: %+v", report.Metrics)
	}
	for _, area := range route {
		if report.PeakOccupancy[area] == 0 {
			t.Fatalf("scenario did not exercise %s", area)
		}
	}
	lastFinished := 0
	for _, event := range report.Events {
		if event.Signal == Finished {
			if event.StudentID != lastFinished+1 {
				t.Fatal("students overtook one another")
			}
			lastFinished = event.StudentID
		}
	}
	if report.Students[5].Waiting[SelfService] == 0 {
		t.Fatal("full downstream area did not create backpressure")
	}
}

func TestStockLimitsAndDiet(t *testing.T) {
	config := DefaultConfig()
	config.MaxStartersPerStudent, config.MaxDessertsPerStudent, config.MaxSelfServiceItems = 2, 2, 3
	config.Menu[0].Portions, config.Menu[1].Portions, config.Menu[3].Portions = 3, 2, 4
	students := testStudents(12)
	for i := range students {
		students[i].StartersWanted, students[i].DessertsWanted = 10, 10
	}
	report := mustRun(t, config, students)
	if report.Stock[0].Taken != 3 || report.Stock[1].Taken != 2 || report.Stock[2].Taken != 0 || report.Stock[3].Taken != 4 {
		t.Fatalf("unexpected stock usage: %+v", report.Stock)
	}
	if report.Metrics.Completed != 12 {
		t.Fatal("stock shortage blocked payment")
	}
	if report.Students[0].Satisfaction <= report.Students[11].Satisfaction {
		t.Fatal("missing courses did not reduce satisfaction")
	}
}

func TestMeatPreferenceAndVegetarianFallback(t *testing.T) {
	config := DefaultConfig()
	config.IndoorLossPerSecond = 0
	config.Menu[2].Portions = 1
	students := testStudents(2)
	for i := range students {
		students[i].Preference = MeatPreferred
	}
	report := mustRun(t, config, students)
	if report.Students[0].Meal.Main.Vegetarian || !report.Students[1].Meal.Main.Vegetarian {
		t.Fatal("meat preference or fallback was ignored")
	}
	if report.Students[0].Satisfaction-report.Students[1].Satisfaction != 13 {
		t.Fatal("preference was not reflected in satisfaction")
	}
}

func TestOutdoorWaitCostsMore(t *testing.T) {
	config := DefaultConfig()
	config.Capacities = Capacities{Outside: 1, Inside: 1, SelfService: 1, MainService: 1, Payment: 1}
	config.MaxDuration = time.Second
	report, err := Run(context.Background(), config, testStudents(3))
	if !errors.Is(err, ErrTimeLimit) {
		t.Fatalf("expected time limit, got %v", err)
	}
	checkInvariants(t, config, report)
	inside, outside := report.Students[1], report.Students[2]
	if inside.Waiting[Inside] != time.Second || outside.Waiting[Outside] != time.Second {
		t.Fatal("unexpected waiting area")
	}
	if math.Abs(inside.Satisfaction-79.92) > 1e-9 || math.Abs(outside.Satisfaction-79.70) > 1e-9 {
		t.Fatalf("unexpected satisfaction: indoor=%f outdoor=%f", inside.Satisfaction, outside.Satisfaction)
	}
}

func TestPatienceAndSatisfactionCauseDepartureBeforeTray(t *testing.T) {
	for _, cause := range []string{"patience", "satisfaction"} {
		t.Run(cause, func(t *testing.T) {
			config := DefaultConfig()
			config.Capacities = Capacities{Outside: 1, Inside: 1, SelfService: 1, MainService: 1, Payment: 1}
			students := testStudents(3)
			for i := range students {
				if cause == "patience" {
					students[i].Patience = time.Second
				} else {
					students[i].Satisfaction = 0.01
				}
			}
			report := mustRun(t, config, students)
			if report.Metrics.Completed != 1 || report.Metrics.Abandoned != 2 {
				t.Fatalf("unexpected departures: %+v", report.Metrics)
			}
			for _, student := range report.Students[1:] {
				if student.ExitTime != time.Second || student.Meal.Main != nil || len(student.Meal.Starters) != 0 {
					t.Fatal("departed student waited too long or consumed inventory")
				}
			}
		})
	}
}

func TestSecondPaymentWindowIncreasesThroughput(t *testing.T) {
	config := DefaultConfig()
	config.MainServiceTime, config.PaymentTime = time.Second, 10*time.Second
	config.Servers = 3
	students := testStudents(15)
	one := mustRun(t, config, students)
	config.Capacities.Payment, config.Cashiers = 2, 2
	two := mustRun(t, config, students)
	if two.Duration >= one.Duration || two.PeakOccupancy[Payment] != 2 {
		t.Fatalf("second window did not improve throughput: %s -> %s", one.Duration, two.Duration)
	}
	if two.Staff[3].Served == 0 || two.Staff[4].Served == 0 {
		t.Fatal("both cashiers should participate")
	}
}

func TestTimeLimitReturnsUnservedReservations(t *testing.T) {
	config := DefaultConfig()
	config.MaxDuration = 5 * time.Second // Sides collected at t=0, main reserved at t=4.
	report, err := Run(context.Background(), config, testStudents(1))
	if !errors.Is(err, ErrTimeLimit) {
		t.Fatalf("expected time limit, got %v", err)
	}
	checkInvariants(t, config, report)
	if report.Stock[1].Remaining != config.Menu[1].Portions || report.Stock[1].Taken != 0 ||
		report.Students[0].Meal.Main != nil || report.Staff[0].BusyTime != time.Second {
		t.Fatal("in-flight main was not returned to stock")
	}
	if report.Stock[0].Taken != 1 || report.Metrics.Cancelled != 1 {
		t.Fatal("already collected sides must remain consumed")
	}
}

func TestCancellationInterruptsPacingAndJoinsAgents(t *testing.T) {
	config := DefaultConfig()
	config.TickInterval = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		report Report
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		report, err := Run(ctx, config, testStudents(10))
		done <- outcome{report: report, err: err}
	}()
	// Cancellation may occur before or during pacing; either must terminate
	// the agents without waiting for the configured hour.
	time.AfterFunc(20*time.Millisecond, cancel)
	select {
	case result := <-done:
		if !errors.Is(result.err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", result.err)
		}
		checkInvariants(t, config, result.report)
		if result.report.Metrics.Cancelled != 10 {
			t.Fatal("not all unfinished students were cancelled")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("simulation failed to cancel promptly")
	}
}

func TestCompletionDoesNotWaitForAnotherPlaybackInterval(t *testing.T) {
	config := DefaultConfig()
	config.SelfServiceTime, config.MainServiceTime, config.PaymentTime = time.Second, time.Second, time.Second
	config.TickInterval = 500 * time.Millisecond
	// Three service ticks need two playback pauses. A redundant third pause
	// would incorrectly turn a fully paid meal into a cancelled student.
	ctx, cancel := context.WithTimeout(context.Background(), 1250*time.Millisecond)
	defer cancel()
	report, err := Run(ctx, config, testStudents(1))
	if err != nil {
		t.Fatalf("completed service was delayed or cancelled: %v", err)
	}
	checkInvariants(t, config, report)
	if report.Duration != 3*time.Second || report.Metrics.Completed != 1 {
		t.Fatal("service should have completed at the third logical tick")
	}
}

func TestRepeatableDespitePlaybackSpeed(t *testing.T) {
	config := DefaultConfig()
	students, err := GenerateStudents(90, 42, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	first := mustRun(t, config, students)
	config.TickInterval = time.Microsecond
	second := mustRun(t, config, students)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("agent scheduling or playback speed changed the result")
	}
}

func TestArrivalOrderAndClockRounding(t *testing.T) {
	config := DefaultConfig()
	config.SelfServiceTime = 1500 * time.Millisecond
	config.MainServiceTime = 2500 * time.Millisecond
	config.PaymentTime = 500 * time.Millisecond
	students := testStudents(2)
	students[0].ArrivalTime, students[1].ArrivalTime = 1500*time.Millisecond, 500*time.Millisecond
	report := mustRun(t, config, students)
	if report.Students[0].ArrivalTime != 2*time.Second || report.Students[1].ArrivalTime != time.Second {
		t.Fatal("arrivals were not ordered and rounded to clock boundaries")
	}
	var completionOrder []int
	for _, event := range report.Events {
		if event.Signal == Finished {
			completionOrder = append(completionOrder, event.StudentID)
		}
	}
	if !reflect.DeepEqual(completionOrder, []int{2, 1}) || report.Duration != 8*time.Second {
		t.Fatalf("unexpected rounded service timing: %v, %s", completionOrder, report.Duration)
	}
}

func TestEmptyPopulation(t *testing.T) {
	report := mustRun(t, DefaultConfig(), nil)
	if report.Duration != 0 || report.Metrics.AverageWait != 0 {
		t.Fatal("empty population should complete immediately")
	}
}

func TestInvalidInputs(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Config, []Student)
	}{
		{"capacity", func(c *Config, _ []Student) { c.Capacities.Payment = 0 }},
		{"workers", func(c *Config, _ []Student) { c.Servers = 0 }},
		{"clock", func(c *Config, _ []Student) { c.Tick = 0 }},
		{"negative stock", func(c *Config, _ []Student) { c.Menu[0].Portions = -1 }},
		{"duplicate dish", func(c *Config, _ []Student) { c.Menu[1].ID = c.Menu[0].ID }},
		{"duplicate student", func(_ *Config, s []Student) { s[1].ID = s[0].ID }},
		{"invalid preference", func(_ *Config, s []Student) { s[0].Preference = "unknown" }},
		{"invalid satisfaction", func(_ *Config, s []Student) { s[0].Satisfaction = math.NaN() }},
		{"outdoor loss", func(c *Config, _ []Student) { c.OutdoorLossPerSecond = c.IndoorLossPerSecond }},
		{"negative allowance", func(c *Config, _ []Student) { c.MaxSelfServiceItems = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config, students := DefaultConfig(), testStudents(2)
			tc.change(&config, students)
			if _, err := Run(context.Background(), config, students); err == nil {
				t.Fatal("invalid input was accepted")
			}
		})
	}
	if _, err := GenerateStudents(-1, 0, time.Second); err == nil {
		t.Fatal("negative population accepted")
	}
	if _, err := GenerateStudents(3, 0, time.Duration(math.MaxInt64)); err == nil {
		t.Fatal("overflowing arrival schedule accepted")
	}
}

// Package simulation models a CROUS lunch service without a graphical interface.
package simulation

import "time"

type Preference string

const (
	Vegetarian    Preference = "vegetarian"
	MeatPreferred Preference = "meat_preferred"
)

type Course string

const (
	Starter Course = "starter"
	Main    Course = "main"
	Dessert Course = "dessert"
)

type Area string

const (
	Outside     Area = "outdoor_queue"
	Inside      Area = "indoor_queue"
	SelfService Area = "self_service"
	MainService Area = "main_service"
	Payment     Area = "payment"
)

var route = [...]Area{Outside, Inside, SelfService, MainService, Payment}

type Status string

const (
	Scheduled Status = "scheduled"
	Active    Status = "active"
	Completed Status = "completed"
	Abandoned Status = "abandoned"
	Rejected  Status = "rejected"
	Cancelled Status = "cancelled"
)

// Signal is an agent interaction or a change in the simulation's state.
type Signal string

const (
	Arrived          Signal = "arrived"
	Entered          Signal = "entered"
	SelfServiceStart Signal = "self_service_started"
	FoodCollected    Signal = "food_collected"
	OrderPlaced      Signal = "order_placed"
	MainServed       Signal = "main_served"
	FoodUnavailable  Signal = "food_unavailable"
	PaymentStarted   Signal = "payment_started"
	Paid             Signal = "paid"
	Finished         Signal = "finished"
	LeftQueue        Signal = "left_queue"
	QueueFull        Signal = "queue_full"
	Stopped          Signal = "stopped"
	Progress         Signal = "progress"
)

type Role string

const (
	Server  Role = "server"
	Cashier Role = "cashier"
)

type Student struct {
	ID             int
	Preference     Preference
	ArrivalTime    time.Duration
	Patience       time.Duration
	Satisfaction   float64
	StartersWanted int
	DessertsWanted int
}

type Staff struct {
	ID          int
	Role        Role
	ServiceTime time.Duration
}

type Dish struct {
	ID         string
	Name       string
	Course     Course
	Vegetarian bool
	Portions   int
}

// Stock tracks daily preparation. Remaining + Reserved + Taken == Dish.Portions.
// A main is reserved when ordered and taken only when the server delivers it.
type Stock struct {
	Dish      Dish
	Remaining int
	Reserved  int
	Taken     int
}

type Meal struct {
	Starters []Dish
	Main     *Dish
	Desserts []Dish
}

type Event struct {
	Time      time.Duration
	Signal    Signal
	StudentID int
	StaffID   int
	From      Area
	To        Area
	DishIDs   []string
}

type StudentResult struct {
	Student      Student
	Status       Status
	Area         Area
	ArrivalTime  time.Duration
	ExitTime     time.Duration
	Satisfaction float64
	TotalWait    time.Duration
	Waiting      map[Area]time.Duration
	Path         []Area
	Meal         Meal
}

type StaffResult struct {
	Staff       Staff
	Served      int
	BusyTime    time.Duration
	Utilization float64
}

type Metrics struct {
	Completed           int
	Abandoned           int
	Rejected            int
	Cancelled           int
	AverageWait         time.Duration
	AverageWaitByArea   map[Area]time.Duration
	AverageSatisfaction float64
	AbandonmentRate     float64
	MealsPerMinute      float64
}

type Report struct {
	Duration      time.Duration
	Students      []StudentResult
	Staff         []StaffResult
	Stock         []Stock
	PeakOccupancy map[Area]int
	Events        []Event
	Metrics       Metrics
}

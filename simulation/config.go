package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

type Capacities struct {
	Outside     int
	Inside      int
	SelfService int
	MainService int
	Payment     int
}

// Config durations are simulated time, except TickInterval (wall-clock pacing).
type Config struct {
	Capacities             Capacities
	Tick                   time.Duration
	TickInterval           time.Duration
	MaxDuration            time.Duration
	SelfServiceTime        time.Duration
	MainServiceTime        time.Duration
	PaymentTime            time.Duration
	Servers                int
	Cashiers               int
	MaxStartersPerStudent  int
	MaxDessertsPerStudent  int
	MaxSelfServiceItems    int
	IndoorLossPerSecond    float64
	OutdoorLossPerSecond   float64
	PreferredMainBonus     float64
	AlternativeMainPenalty float64
	MissingMainPenalty     float64
	MissingSidePenalty     float64
	CourseVarietyBonus     float64
	Menu                   []Dish
}

func DefaultConfig() Config {
	return Config{
		Capacities:             Capacities{Outside: 50, Inside: 20, SelfService: 6, MainService: 3, Payment: 1},
		Tick:                   time.Second,
		MaxDuration:            15 * time.Minute,
		SelfServiceTime:        4 * time.Second,
		MainServiceTime:        8 * time.Second,
		PaymentTime:            4 * time.Second,
		Servers:                2,
		Cashiers:               1,
		MaxStartersPerStudent:  1,
		MaxDessertsPerStudent:  1,
		MaxSelfServiceItems:    2,
		IndoorLossPerSecond:    0.08,
		OutdoorLossPerSecond:   0.30,
		PreferredMainBonus:     3,
		AlternativeMainPenalty: 10,
		MissingMainPenalty:     25,
		MissingSidePenalty:     5,
		CourseVarietyBonus:     2,
		Menu: []Dish{
			{ID: "salad", Name: "Salade", Course: Starter, Vegetarian: true, Portions: 90},
			{ID: "vegetable_curry", Name: "Curry de legumes", Course: Main, Vegetarian: true, Portions: 40},
			{ID: "chicken", Name: "Poulet et riz", Course: Main, Portions: 60},
			{ID: "fruit", Name: "Fruit", Course: Dessert, Vegetarian: true, Portions: 90},
		},
	}
}

func (c Config) validate(students []Student) error {
	for _, item := range []struct {
		name  string
		value int
	}{
		{"outside capacity", c.Capacities.Outside}, {"inside capacity", c.Capacities.Inside},
		{"self-service capacity", c.Capacities.SelfService}, {"main-service capacity", c.Capacities.MainService},
		{"payment capacity", c.Capacities.Payment}, {"servers", c.Servers}, {"cashiers", c.Cashiers},
	} {
		if item.value <= 0 {
			return fmt.Errorf("%s must be positive", item.name)
		}
	}
	for _, item := range []struct {
		name  string
		value time.Duration
	}{
		{"tick", c.Tick}, {"maximum duration", c.MaxDuration}, {"self-service time", c.SelfServiceTime},
		{"main-service time", c.MainServiceTime}, {"payment time", c.PaymentTime},
	} {
		if item.value <= 0 {
			return fmt.Errorf("%s must be positive", item.name)
		}
	}
	if c.TickInterval < 0 {
		return fmt.Errorf("tick interval cannot be negative")
	}
	if c.MaxStartersPerStudent < 0 || c.MaxDessertsPerStudent < 0 || c.MaxSelfServiceItems < 0 {
		return fmt.Errorf("self-service limits cannot be negative")
	}
	for _, value := range []float64{
		c.IndoorLossPerSecond, c.OutdoorLossPerSecond, c.PreferredMainBonus,
		c.AlternativeMainPenalty, c.MissingMainPenalty, c.MissingSidePenalty, c.CourseVarietyBonus,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("satisfaction adjustments must be finite and nonnegative")
		}
	}
	if c.OutdoorLossPerSecond <= c.IndoorLossPerSecond {
		return fmt.Errorf("outdoor satisfaction loss must be greater than indoor loss")
	}
	dishIDs := make(map[string]bool)
	for _, dish := range c.Menu {
		if dish.ID == "" || dishIDs[dish.ID] {
			return fmt.Errorf("dish IDs must be nonempty and unique: %q", dish.ID)
		}
		dishIDs[dish.ID] = true
		if dish.Course != Starter && dish.Course != Main && dish.Course != Dessert {
			return fmt.Errorf("invalid course for dish %q", dish.ID)
		}
		if dish.Portions < 0 {
			return fmt.Errorf("negative portions for dish %q", dish.ID)
		}
	}
	studentIDs := make(map[int]bool)
	for _, student := range students {
		if student.ID <= 0 || studentIDs[student.ID] {
			return fmt.Errorf("student IDs must be positive and unique: %d", student.ID)
		}
		studentIDs[student.ID] = true
		if student.Preference != Vegetarian && student.Preference != MeatPreferred {
			return fmt.Errorf("invalid preference for student %d", student.ID)
		}
		if student.ArrivalTime < 0 || student.Patience < 0 || student.StartersWanted < 0 || student.DessertsWanted < 0 {
			return fmt.Errorf("negative student attributes for student %d", student.ID)
		}
		if math.IsNaN(student.Satisfaction) || math.IsInf(student.Satisfaction, 0) ||
			student.Satisfaction < 0 || student.Satisfaction > 100 {
			return fmt.Errorf("satisfaction must be between 0 and 100 for student %d", student.ID)
		}
	}
	return nil
}

// GenerateStudents produces repeatable profiles without using a shared RNG.
func GenerateStudents(count int, seed int64, arrivalInterval time.Duration) ([]Student, error) {
	if count < 0 || arrivalInterval < 0 {
		return nil, fmt.Errorf("student count and arrival interval cannot be negative")
	}
	if count > 1 && int64(arrivalInterval) > math.MaxInt64/int64(count-1) {
		return nil, fmt.Errorf("arrival schedule overflows time.Duration")
	}
	rng := rand.New(rand.NewSource(seed))
	students := make([]Student, count)
	for i := range students {
		preference := MeatPreferred
		if rng.Float64() < 0.35 {
			preference = Vegetarian
		}
		students[i] = Student{
			ID: i + 1, Preference: preference, ArrivalTime: time.Duration(i) * arrivalInterval,
			Patience:     time.Duration(40+rng.Intn(121)) * time.Second,
			Satisfaction: float64(80 + rng.Intn(16)), StartersWanted: 1, DessertsWanted: 1,
		}
	}
	return students, nil
}

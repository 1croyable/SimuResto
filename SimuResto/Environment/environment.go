// Package environment defines the map and the current simulation state.
package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

const (
	Width  = 1000
	Height = 700
)

type Region struct {
	X, Y, Width, Height float32
}

// the engine updates it and the client draws it.
type Environment struct {
	Settings   Settings
	TimePassed time.Duration // Total simulated time since starting
	Paused     bool

	Roads         []Region
	FoodRegion    Region
	CashierRegion Region
	DiningRegion  Region
	Route         []agents.Point
	CheckoutRoute []agents.Point
	DiningRoute   []agents.Point

	Student agents.StudentState
	Cook    agents.StaffState
	Cashier agents.StaffState
	Dish    agents.Dish

	RemainingPortions int
}

func New(settings Settings) *Environment {
	if settings.FramesPerSecond <= 0 || settings.TicksPerSecond <= 0 || settings.WalkingSpeed <= 0 || settings.PaymentTime < 0 {
		panic("invalid settings")
	}

	world := &Environment{
		Settings:      settings,
		Roads:         []Region{{35, 110, 965, 60}, {35, 110, 60, 440}, {35, 490, 845, 60}},
		FoodRegion:    Region{155, 550, 555, 150},
		CashierRegion: Region{763, 550, 133, 150},
		DiningRegion:  Region{705, 340, 180, 125},
		Route:         []agents.Point{{X: 980, Y: 140}, {X: 65, Y: 140}, {X: 65, Y: 520}, {X: 430, Y: 520}},
		CheckoutRoute: []agents.Point{{X: 825, Y: 520}},
		DiningRoute:   []agents.Point{{X: 825, Y: 415}},
		Cook: agents.StaffState{
			Staff:    agents.Staff{ID: 1, Role: agents.Server, ServiceTime: settings.ServiceTime},
			Position: agents.Point{X: 430, Y: 585},
		},
		Cashier: agents.StaffState{
			Staff:    agents.Staff{ID: 2, Role: agents.Cashier, ServiceTime: settings.PaymentTime},
			Position: agents.Point{X: 825, Y: 585},
		},
		Dish: agents.Dish{ID: "main", Name: "Vegetable dish", Course: agents.Main,
			Vegetarian: true, Portions: 1, SatisfactionScore: 70},
		RemainingPortions: 1,
	}

	world.Student = agents.StudentState{
		Student:  agents.Student{ID: 1, Preference: agents.Vegetarian},
		Position: world.Route[0], NextWaypoint: 1, Status: agents.Walking,
	}
	return world
}

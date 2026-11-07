// Package environment defines the map and the current simulation state.
package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

const (
	Width  = 1200
	Height = 700
)

type Region struct {
	X, Y, Width, Height float32
}

type DiningArea struct {
	Name         string
	Region       Region
	Route        []agents.Point
	CleanupRoute []agents.Point
}

// the engine updates it and the client draws it.
type Environment struct {
	Settings   Settings
	TimePassed time.Duration // Total simulated time since starting
	Paused     bool

	Roads         []Region
	FoodRegion    Region
	CashierRegion Region
	DiningAreas   [3]DiningArea
	WasteRegion   Region

	Route         []agents.Point
	CheckoutRoute []agents.Point
	ExitRoute     []agents.Point

	Student agents.StudentState
	Cook    agents.StaffState
	Cashier agents.StaffState
	Dish    agents.Dish

	RemainingPortions int
}

func New(settings Settings) *Environment {
	if settings.FramesPerSecond <= 0 || settings.TicksPerSecond <= 0 || settings.WalkingSpeed <= 0 || settings.PaymentTime < 0 || settings.DiningTime < 0 {
		panic("invalid settings")
	}

	const lowerRoadY = 520
	const diningAisleX = 770
	wasteStop := agents.Point{X: 930, Y: 600}

	world := &Environment{
		Settings:      settings,
		Roads:         []Region{{35, 60, Width - 35, 60}, {35, 60, 60, 490}, {35, 490, Width - 35, 60}},
		FoodRegion:    Region{130, 550, 435, 150},
		CashierRegion: Region{610, 550, 125, 150},
		DiningAreas: [3]DiningArea{
			{Name: "Window dining", Region: Region{180, 120, 820, 55}},
			{Name: "Left dining", Region: Region{130, 200, 605, 265}},
			{Name: "Right dining", Region: Region{805, 200, 335, 265}},
		},
		WasteRegion:   Region{805, 550, 270, 120},
		Route:         []agents.Point{{X: Width - 20, Y: 90}, {X: 65, Y: 90}, {X: 65, Y: lowerRoadY}, {X: 350, Y: lowerRoadY}},
		CheckoutRoute: []agents.Point{{X: 670, Y: lowerRoadY}},
		ExitRoute:     []agents.Point{{X: wasteStop.X, Y: lowerRoadY}, {X: Width, Y: lowerRoadY}},
		Cook: agents.StaffState{
			Staff:    agents.Staff{ID: 1, Role: agents.Server, ServiceTime: settings.ServiceTime},
			Position: agents.Point{X: 350, Y: 585},
		},
		Cashier: agents.StaffState{
			Staff:    agents.Staff{ID: 2, Role: agents.Cashier, ServiceTime: settings.PaymentTime},
			Position: agents.Point{X: 670, Y: 585},
		},
		Dish: agents.Dish{ID: "main", Name: "Vegetable dish", Course: agents.Main,
			Vegetarian: true, Portions: 1, SatisfactionScore: 70},
		RemainingPortions: 1,
	}

	aisleEntrance := agents.Point{X: diningAisleX, Y: lowerRoadY}
	for i := range world.DiningAreas {
		area := &world.DiningAreas[i]
		region := area.Region
		seat := agents.Point{X: float64(region.X + region.Width/2), Y: float64(region.Y + region.Height/2)}
		aisleTurn := agents.Point{X: diningAisleX, Y: seat.Y}
		area.Route = []agents.Point{aisleEntrance, aisleTurn, seat}
		area.CleanupRoute = []agents.Point{aisleTurn, aisleEntrance, {X: wasteStop.X, Y: lowerRoadY}, wasteStop}
	}

	world.Student = agents.StudentState{
		Student:  agents.StudentData{ID: 1},
		Position: world.Route[0], NextWaypoint: 1, Status: agents.Walking,
	}
	return world
}

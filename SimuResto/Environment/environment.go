package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	model "simuresto/SimuResto/Shared/Model"
)

const (
	Width  = 1200
	Height = 700
)

type Region struct {
	X, Y, Width, Height float32
}

type SauceStation struct {
	Queue    *model.Queue
	Portions int
	FreeAt   time.Duration // Simulation time when the last pickup ended.
}

// the engine updates it and the client draws it.
type Environment struct {
	Settings       Settings
	TimePassed     time.Duration // Total simulated time since starting
	OperatingPhase OperatingPhase

	Roads               []Region
	StarterRegions      [2]Region
	MainRegion          Region
	CashierRegion       Region
	Cashier2Region      Region
	WasteRegion         Region
	WaterAndSauceRegion Region

	Dining     DiningLayout
	SeatOwners []int // Indexed by seat ID - 1. A zero owner == the seat is free.
	SeatQueue  *model.Queue

	Starter1Route  []agents.Point
	Starter2Route  []agents.Point
	MainRoute      []agents.Point
	Checkout1Route []agents.Point
	ExitRoute      []agents.Point

	Checkout2Route []agents.Point

	StudentStates []agents.StudentState
	StudentsByID  map[int]*agents.StudentState
	Cook          agents.StaffState
	Cashiers      [MaxCashiers]agents.StaffState

	Menu     []agents.Dish
	Counters [3][]CounterStock // Starter 1, Starter 2, Main
	Restock  RestockState
	Sauce    SauceStation

	Queue *model.Queue

	Statistics Statistics

	SimulationSpeed float64 // Simulated seconds per real second.
}

type OperatingPhase string

const (
	OperatingOpen    OperatingPhase = "OPEN"
	OperatingClosing OperatingPhase = "CLOSING"
	OperatingBreak   OperatingPhase = "BREAK"
)

func New(settings Settings) *Environment {
	const (
		laneY          = Height - 108
		laneTop        = laneY - 30
		laneBottom     = laneY + 30
		lowerPickupY   = laneBottom - 10
		upperPaymentY  = laneTop + 10
		deskHeight     = 60
		upperDeskY     = laneTop - deskHeight
		lowerZoneDepth = Height - laneBottom - 12
		diningAisleX   = 770
		diningTop      = 200
		diningHeight   = laneTop - 25 - diningTop
	)
	wasteStop := agents.Point{X: 930, Y: laneBottom + lowerZoneDepth/2}

	world := &Environment{
		Settings:       settings,
		OperatingPhase: OperatingBreak,

		Roads: []Region{{35, 60, Width - 35, 60}, {35, 60, 60, laneBottom - 60}, {35, laneTop, Width - 35, laneBottom - laneTop}},
		StarterRegions: [2]Region{
			{138, laneBottom, 74, 34},
			{238, laneBottom, 74, 34},
		},
		MainRegion:          Region{344, laneBottom, 207, lowerZoneDepth},
		CashierRegion:       Region{620, laneBottom, 105, deskHeight},
		Cashier2Region:      Region{620, upperDeskY, 105, deskHeight},
		WasteRegion:         Region{805, laneBottom, 270, lowerZoneDepth},
		WaterAndSauceRegion: Region{468, upperDeskY, 50, 36},

		// Entrance → Starter 1
		Starter1Route: []agents.Point{
			{X: Width - 20, Y: 90},
			{X: 65, Y: 90},
			{X: 65, Y: 140}, // door
			{X: 65, Y: lowerPickupY},
			{X: 175, Y: lowerPickupY},
		},

		Checkout2Route: []agents.Point{
			{X: 540, Y: lowerPickupY},
			{X: 560, Y: lowerPickupY - 2},
			{X: 578, Y: lowerPickupY - 10},
			{X: 602, Y: upperPaymentY + 8},
			{X: 620, Y: upperPaymentY},
			{X: 670, Y: upperPaymentY},
		},

		// Starter 1 → Starter 2
		Starter2Route: []agents.Point{{X: 275, Y: lowerPickupY}},

		// Starter 2 → Main course
		MainRoute: []agents.Point{{X: 447.5, Y: lowerPickupY}},

		// Main course → Cashier
		Checkout1Route: []agents.Point{{X: 670, Y: lowerPickupY}},

		ExitRoute: []agents.Point{{X: wasteStop.X, Y: laneY}, {X: Width, Y: laneY}},

		Cook: agents.StaffState{
			Staff:    agents.StaffData{ID: 1, Role: agents.Server, ServiceTime: settings.ServiceTime},
			Position: agents.Point{X: 447.5, Y: laneBottom + lowerZoneDepth/2},
		},

		Cashiers: [MaxCashiers]agents.StaffState{
			{
				Staff: agents.StaffData{
					ID:          2,
					Role:        agents.Cashier,
					ServiceTime: settings.PaymentTime,
				},
				Position: agents.Point{X: 645, Y: laneBottom + deskHeight/2},
			},
			{
				Staff: agents.StaffData{
					ID:          3,
					Role:        agents.Cashier,
					ServiceTime: settings.PaymentTime,
				},
				Position: agents.Point{X: 645, Y: upperDeskY + deskHeight/2},
			},
		},

		Menu: []agents.Dish{
			{
				ID: "salad", Name: "Green salad",
				Course:     agents.Starter,
				Vegetarian: true,
				Portions:   0,

				RefillBatch:    10,
				ColorScore:     0.9,
				TasteScore:     0.7,
				SatietyScore:   0.3,
				StarterCounter: 1,
				MaxPortions:    40,
			},
			{
				ID:         "soup",
				Name:       "Vegetable soup",
				Course:     agents.Starter,
				Vegetarian: true,
				Portions:   0,

				RefillBatch:    10,
				ColorScore:     0.6,
				TasteScore:     0.9,
				SatietyScore:   0.7,
				StarterCounter: 2,
				MaxPortions:    40,
			},
			{
				ID:         "vegetable",
				Name:       "Vegetable dish",
				Course:     agents.Main,
				Vegetarian: true,
				Portions:   0,

				RefillBatch:  10,
				ColorScore:   0.7,
				TasteScore:   0.8,
				SatietyScore: 0.7,
				MaxPortions:  30,
			},
			{
				ID:         "pasta",
				Name:       "Pasta",
				Course:     agents.Main,
				Vegetarian: true,
				Portions:   0,

				RefillBatch:  10,
				ColorScore:   0.8,
				TasteScore:   0.9,
				SatietyScore: 0.9,
				MaxPortions:  30,
			},
		},

		SimulationSpeed: 10.0,
	}

	areas := [3]Region{
		{180, 120, 820, 55},
		{130, diningTop, 605, diningHeight},
		{805, diningTop, 335, diningHeight},
	}
	world.Dining = newDiningLayout(world, areas, diningAisleX, wasteStop)
	world.SeatOwners = make([]int, len(world.Dining.Seats))
	world.SeatQueue, _ = model.NewQueue(max(1, settings.StudentCount))

	world.InitCounters()

	world.Queue, _ = model.NewQueue(settings.StudentCount)
	world.Sauce.Queue, _ = model.NewQueue(settings.SauceCapacity)
	world.Sauce.Portions = settings.SaucePortions

	world.StudentStates = make([]agents.StudentState, 0, settings.StudentCount)

	world.StudentsByID = make(map[int]*agents.StudentState, settings.StudentCount)

	return world
}

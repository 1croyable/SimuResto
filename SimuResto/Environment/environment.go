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

type DiningArea struct {
	Name              string
	Region            Region
	Route             []agents.Point
	UpperCashierRoute []agents.Point
	CleanupRoute      []agents.Point
}

// the engine updates it and the client draws it.
type Environment struct {
	Settings       Settings
	TimePassed     time.Duration // Total simulated time since starting
	OperatingPhase OperatingPhase

	Roads          []Region
	StarterRegions [2]Region
	MainRegion     Region
	CashierRegion  Region
	Cashier2Region Region
	WasteRegion    Region

	DiningAreas [3]DiningArea

	Route         []agents.Point
	Starter2Route []agents.Point
	MainRoute     []agents.Point
	CheckoutRoute []agents.Point
	ExitRoute     []agents.Point

	Checkout2Route    []agents.Point
	Cashier2ExitRoute []agents.Point

	StudentStates []agents.StudentState
	StudentsByID  map[int]*agents.StudentState
	Cook          agents.StaffState
	Cashiers      [MaxCashiers]agents.StaffState

	Menu []agents.Dish

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
	const lowerRoadY = 520
	const diningAisleX = 770
	wasteStop := agents.Point{X: 930, Y: 600}

	world := &Environment{
		Settings:       settings,
		OperatingPhase: OperatingBreak,

		Roads: []Region{{35, 60, Width - 35, 60}, {35, 60, 60, 490}, {35, 490, Width - 35, 60}},
		StarterRegions: [2]Region{
			{130, 570, 90, 45},
			{230, 570, 90, 45},
		},
		MainRegion:     Region{330, 550, 235, 150},
		CashierRegion:  Region{610, 550, 125, 150},
		Cashier2Region: Region{610, 405, 125, 40},
		DiningAreas: [3]DiningArea{
			{Name: "Window dining", Region: Region{180, 120, 820, 55}},
			{Name: "Left dining", Region: Region{130, 200, 605, 265}},
			{Name: "Right dining", Region: Region{805, 200, 335, 265}},
		},
		WasteRegion: Region{805, 550, 270, 120},

		// Entrance → Starter 1
		Route: []agents.Point{
			{X: Width - 20, Y: 90},
			{X: 65, Y: 90},
			{X: 65, Y: 140}, // door
			{X: 65, Y: lowerRoadY},
			{X: 175, Y: lowerRoadY},
		},

		Checkout2Route: []agents.Point{
			{X: 540, Y: 520},
			{X: 558, Y: 518},
			{X: 571, Y: 511},
			{X: 579, Y: 497},
			{X: 588, Y: 486},
			{X: 602, Y: 477},
			{X: 620, Y: 472},
			{X: 645, Y: 470},
			{X: 670, Y: 470},
		},

		Cashier2ExitRoute: []agents.Point{
			{X: 770, Y: 470},
		},

		// Starter 1 → Starter 2
		Starter2Route: []agents.Point{{X: 275, Y: lowerRoadY}},

		// Starter 2 → Main course
		MainRoute: []agents.Point{{X: 447.5, Y: lowerRoadY}},

		// Main course → Cashier
		CheckoutRoute: []agents.Point{{X: 670, Y: lowerRoadY}},

		ExitRoute: []agents.Point{{X: wasteStop.X, Y: lowerRoadY}, {X: Width, Y: lowerRoadY}},

		Cook: agents.StaffState{
			Staff:    agents.StaffData{ID: 1, Role: agents.Server, ServiceTime: settings.ServiceTime},
			Position: agents.Point{X: 447.5, Y: 625},
		},

		Cashiers: [MaxCashiers]agents.StaffState{
			{
				Staff: agents.StaffData{
					ID:          2,
					Role:        agents.Cashier,
					ServiceTime: settings.PaymentTime,
				},
				Position: agents.Point{X: 645, Y: 585},
			},
			{
				Staff: agents.StaffData{
					ID:          3,
					Role:        agents.Cashier,
					ServiceTime: settings.PaymentTime,
				},
				Position: agents.Point{X: 645, Y: 425},
			},
		},

		Menu: []agents.Dish{
			{ID: "salad", Name: "Green salad", Course: agents.Starter, Vegetarian: true,
				Portions: settings.StudentCount, ColorScore: 0.9, TasteScore: 0.7, SatietyScore: 0.3, StarterCounter: 1},

			{ID: "soup", Name: "Vegetable soup", Course: agents.Starter, Vegetarian: true,
				Portions: settings.StudentCount, ColorScore: 0.6, TasteScore: 0.9, SatietyScore: 0.7, StarterCounter: 2},

			{ID: "vegetable", Name: "Vegetable dish", Course: agents.Main, Vegetarian: true,
				Portions: (settings.StudentCount + 1) / 2, ColorScore: 0.7, TasteScore: 0.8, SatietyScore: 0.7},

			{ID: "pasta", Name: "Pasta", Course: agents.Main, Vegetarian: true,
				Portions: settings.StudentCount / 2, ColorScore: 0.8, TasteScore: 0.9, SatietyScore: 0.9},
		},

		SimulationSpeed: 1.0,
	}

	aisleEntrance := agents.Point{X: diningAisleX, Y: lowerRoadY}
	for i := range world.DiningAreas {
		area := &world.DiningAreas[i]
		region := area.Region
		seat := agents.Point{X: float64(region.X + region.Width/2), Y: float64(region.Y + region.Height/2)}
		aisleTurn := agents.Point{X: diningAisleX, Y: seat.Y}

		area.Route = []agents.Point{aisleEntrance, aisleTurn, seat}

		upperEntrance := agents.Point{X: diningAisleX, Y: 470}
		area.UpperCashierRoute = []agents.Point{upperEntrance, aisleTurn, seat}

		area.CleanupRoute = []agents.Point{aisleTurn, aisleEntrance, {X: wasteStop.X, Y: lowerRoadY}, wasteStop}
	}

	world.Queue, _ = model.NewQueue(settings.StudentCount)

	world.StudentStates = make([]agents.StudentState, 0, settings.StudentCount)

	world.StudentsByID = make(map[int]*agents.StudentState, settings.StudentCount)

	return world
}

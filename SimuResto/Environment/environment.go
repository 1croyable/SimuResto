package environment

import (
	"time"

	agents "simuresto/SimuResto/Agents"
	food "simuresto/SimuResto/Shared/Food"
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

	Roads          []Region
	StarterRegions [2]Region
	MainRegion     Region
	CashierRegion  Region
	DiningAreas    [3]DiningArea
	WasteRegion    Region

	Route         []agents.Point
	Starter2Route []agents.Point
	MainRoute     []agents.Point
	CheckoutRoute []agents.Point
	ExitRoute     []agents.Point

	Students     []agents.StudentState
	StudentsByID map[int]*agents.StudentState

	Queue *model.Queue

	Cook    agents.StaffState
	Cashier agents.StaffState

	Menu              []food.Dish
	RemainingPortions int
}

func New(settings Settings) *Environment {
	const lowerRoadY = 520
	const diningAisleX = 770
	wasteStop := agents.Point{X: 930, Y: 600}

	world := &Environment{
		Settings: settings,
		Roads:    []Region{{35, 60, Width - 35, 60}, {35, 60, 60, 490}, {35, 490, Width - 35, 60}},
		StarterRegions: [2]Region{
			{130, 570, 90, 45},
			{230, 570, 90, 45},
		},
		MainRegion:    Region{330, 550, 235, 150},
		CashierRegion: Region{610, 550, 125, 150},
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
			{X: 65, Y: lowerRoadY},
			{X: 175, Y: lowerRoadY},
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
		Cashier: agents.StaffState{
			Staff:    agents.StaffData{ID: 2, Role: agents.Cashier, ServiceTime: settings.PaymentTime},
			Position: agents.Point{X: 670, Y: 585},
		},
		Menu: []food.Dish{
			{ID: "vegetable", Name: "Vegetable dish", Course: food.Main, Vegetarian: true, Portions: (settings.StudentCount + 1) / 2},
			{ID: "pasta", Name: "Pasta", Course: food.Main, Vegetarian: true, Portions: settings.StudentCount / 2},
		},
		RemainingPortions: settings.StudentCount,
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

	world.Queue, _ = model.NewQueue(settings.StudentCount)
	world.Students = make([]agents.StudentState, settings.StudentCount)
	world.StudentsByID = make(map[int]*agents.StudentState, settings.StudentCount)
	for i := range world.Students {
		world.Students[i] = agents.NewStudentState(agents.StudentData{
			ID: i + 1, ArrivalTime: time.Duration(i) * settings.ArrivalInterval, Patience: settings.BasePatience,
		}, settings.Seed)
		world.StudentsByID[world.Students[i].Student.ID] = &world.Students[i]
	}
	return world
}

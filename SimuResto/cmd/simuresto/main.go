package main

import (
	"image/color"
	"log"
	"time"

	agents "simuresto/SimuResto/Agents"
	client "simuresto/SimuResto/Client"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	Simulation    *engine.Game
	Settings      environment.Settings
	Mode          client.ScreenMode
	tickRemainder int64
	showStats     bool
	frameLimiter  client.FrameLimiter
}

func (g *Game) Update() error {
	// process mouse button
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	x, y := ebiten.CursorPosition()

	// Toggle statistics overlay
	if g.Mode != client.Home && inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.showStats = !g.showStats
	}

	switch g.Mode {
	case client.Home:
		if clicked {
			if client.MenuButtonHit(x, y, client.Button1Y) {
				g.startSimulation() // create the world
			} else if client.MenuButtonHit(x, y, client.Button2Y) {
				return ebiten.Termination
			}
		}
		return nil

	case client.Paused:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.Mode = client.Running
			return nil
		}

		if clicked {
			if client.MenuButtonHit(x, y, client.Button1Y) {
				// Resume
				g.Mode = client.Running
			} else if client.MenuButtonHit(x, y, client.Button2Y) {
				// Restart
				g.startSimulation()
			} else if client.MenuButtonHit(x, y, client.Button3Y) {
				g.closeSimulation()
				g.tickRemainder = 0
				g.Mode = client.Home
			}
		}
		return nil
	}

	// pause
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Mode = client.Paused
		return nil
	}

	// Advance simulated time
	speed := g.Simulation.World.SimulationSpeed

	g.tickRemainder += int64(float64(time.Second) * speed)

	rate := int64(g.Settings.TicksPerSecond)
	dt := time.Duration(g.tickRemainder / rate)
	g.tickRemainder %= rate

	// update
	g.Simulation.Step(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.frameLimiter.Wait(g.Settings.FramesPerSecond)

	screen.Fill(color.White)

	if g.Mode == client.Home {
		client.DrawMenu(screen, client.Home)
		return
	}

	// Read the cached snapshot from Engine.
	snapshot := g.Simulation.Snapshot()
	layout := snapshot.Layout

	roadColor := color.RGBA{R: 96, G: 96, B: 96, A: 255}
	foodColor := color.RGBA{R: 224, G: 224, B: 224, A: 255}
	starterColor := color.RGBA{R: 180, G: 220, B: 180, A: 255}

	// Starter counters
	for _, region := range layout.StarterRegions {
		vector.FillRect(screen, region.X, region.Y, region.Width, region.Height, starterColor, false)
	}

	// Main course counter
	vector.FillRect(screen, layout.MainRegion.X, layout.MainRegion.Y, layout.MainRegion.Width, layout.MainRegion.Height, foodColor, false)

	// Cashier counter
	vector.FillRect(screen, layout.Cashier1Region.X, layout.Cashier1Region.Y, layout.Cashier1Region.Width, layout.Cashier1Region.Height, foodColor, false)

	if snapshot.Settings.Cashiers >= 2 {
		vector.FillRect(screen, layout.Cashier2Region.X, layout.Cashier2Region.Y, layout.Cashier2Region.Width, layout.Cashier2Region.Height, foodColor, false)
	}

	// Dining areas
	for _, area := range layout.DiningAreas {
		region := area.Region
		vector.FillRect(screen, region.X, region.Y, region.Width, region.Height, color.RGBA{R: 224, G: 234, B: 228, A: 255}, false)
	}

	// Waste area
	vector.FillRect(screen, layout.WasteRegion.X, layout.WasteRegion.Y, layout.WasteRegion.Width, layout.WasteRegion.Height, color.RGBA{R: 237, G: 225, B: 210, A: 255}, false)

	// Roads
	for _, road := range layout.Roads {
		vector.FillRect(screen, road.X, road.Y, road.Width, road.Height, roadColor, false)
	}

	if snapshot.Settings.Cashiers >= 2 {
		points := append([]agents.Point(nil), layout.Checkout2Route...)

		points = append(points, layout.Cashier2ExitRoute...)

		for i := 1; i < len(points); i++ {
			from := points[i-1]
			to := points[i]

			vector.StrokeLine(screen, float32(from.X), float32(from.Y), float32(to.X), float32(to.Y), 40, roadColor, true)
		}
	}

	ebitenutil.DebugPrintAt(
		screen, "Exit >", environment.Width-58, 512,
	)

	// Staff
	for _, worker := range snapshot.Staff {
		fill := color.RGBA{R: 225, G: 140, B: 45, A: 255}

		// Idle cashier is displayed in gray.
		if worker.Staff.Role == agents.Cashier &&
			worker.StudentID == 0 {
			fill = color.RGBA{R: 120, G: 120, B: 120, A: 255}
		}

		vector.FillCircle(screen, float32(worker.Position.X), float32(worker.Position.Y), 15, fill, true)
	}

	// Students
	for _, student := range snapshot.Students {
		fill := color.RGBA{R: 65, G: 140, B: 220, A: 255}

		switch student.Status {
		case agents.Finished, agents.Scheduled:
			continue

		case agents.Abandoned:
			fill = color.RGBA{R: 140, G: 140, B: 140, A: 255}

			if !student.IsLeaving {
				continue
			}
		}

		if student.Status == agents.Serving ||
			student.Status == agents.Paying {
			fill = color.RGBA{R: 225, G: 140, B: 45, A: 255}
		}

		vector.FillCircle(screen, float32(student.Position.X), float32(student.Position.Y), 6, fill, true)

		// Show a green indicator when carrying a main course.
		isWalkingToExit := student.Status == agents.Walking &&
			student.Target == agents.TargetExit

		if student.Meal.Main != nil && !isWalkingToExit {
			vector.FillCircle(screen, float32(student.Position.X)+8, float32(student.Position.Y), 3, color.RGBA{R: 90, G: 190, B: 90, A: 255}, true)
		}
	}

	// F3 statistics overlay
	if g.showStats {
		client.DrawStats(screen, snapshot)
	}

	// Pause menu overlay
	if g.Mode == client.Paused {
		client.DrawMenu(screen, client.Paused)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return environment.Width, environment.Height
}

func (g *Game) startSimulation() {
	g.closeSimulation()

	world := environment.New(g.Settings)

	g.Simulation = engine.NewGame(world)

	g.tickRemainder = 0
	g.Mode = client.Running
}

func (g *Game) closeSimulation() {
	if g.Simulation != nil {
		g.Simulation.Close()
		g.Simulation = nil
	}
	g.tickRemainder = 0
}

func main() {
	settings := environment.DefaultSettings()
	game := &Game{
		Settings: settings,
		Mode:     client.Home,
	}
	ebiten.SetWindowSize(environment.Width, environment.Height)
	ebiten.SetWindowTitle("SimuResto")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(settings.TicksPerSecond)
	ebiten.SetVsyncEnabled(false)

	err := ebiten.RunGame(game)
	game.closeSimulation()
	if err != nil {
		log.Fatal(err)
	}
}

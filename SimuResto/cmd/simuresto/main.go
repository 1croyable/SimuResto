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
				g.Simulation.World = environment.New(g.Settings)
				g.tickRemainder = 0
				g.Mode = client.Running
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

	g.tickRemainder += int64(time.Second)
	rate := int64(g.Settings.TicksPerSecond)
	dt := time.Duration(g.tickRemainder / rate)
	g.tickRemainder %= rate

	// update
	g.Simulation.Step(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Frame pacing belongs to the client; the engine only receives simulated time.
	g.frameLimiter.Wait(g.Settings.FramesPerSecond)

	screen.Fill(color.White)
	if g.Mode == client.Home {
		client.DrawMenu(screen, client.Home)
		return
	}

	world := g.Simulation.World

	roadColor := color.RGBA{R: 96, G: 96, B: 96, A: 255}
	foodColor := color.RGBA{R: 224, G: 224, B: 224, A: 255}
	starterColor := color.RGBA{R: 180, G: 220, B: 180, A: 255}

	for _, region := range world.StarterRegions {
		vector.FillRect(screen, region.X, region.Y, region.Width, region.Height, starterColor, false)
	}

	vector.FillRect(screen, world.MainRegion.X, world.MainRegion.Y, world.MainRegion.Width, world.MainRegion.Height, foodColor, false)
	vector.FillRect(screen, world.CashierRegion.X, world.CashierRegion.Y, world.CashierRegion.Width, world.CashierRegion.Height, foodColor, false)

	for _, area := range world.DiningAreas {
		vector.FillRect(screen, area.Region.X, area.Region.Y, area.Region.Width, area.Region.Height, color.RGBA{R: 224, G: 234, B: 228, A: 255}, false)
	}

	vector.FillRect(screen, world.WasteRegion.X, world.WasteRegion.Y, world.WasteRegion.Width, world.WasteRegion.Height, color.RGBA{R: 237, G: 225, B: 210, A: 255}, false)

	for _, road := range world.Roads {
		vector.FillRect(screen, road.X, road.Y, road.Width, road.Height, roadColor, false)
	}

	ebitenutil.DebugPrintAt(screen, "Exit >", environment.Width-58, 512)

	cook := world.Cook.Position
	vector.FillCircle(screen, float32(cook.X), float32(cook.Y), 15,
		color.RGBA{R: 225, G: 140, B: 45, A: 255}, true)
	cashierColor := color.RGBA{R: 120, G: 120, B: 120, A: 255}
	if world.Cashier.StudentID != 0 {
		cashierColor = color.RGBA{R: 225, G: 140, B: 45, A: 255}
	}

	register := world.Cashier.Position
	vector.FillCircle(screen, float32(register.X), float32(register.Y), 15, cashierColor, true)

	finished, abandoned := 0, 0
	for _, student := range world.StudentStates {
		if student.Status == agents.Finished {
			finished++
			continue
		}
		if student.Status == agents.Abandoned {
			abandoned++
			continue
		}
		if student.Status == agents.Scheduled {
			continue
		}
		fill := color.RGBA{R: 65, G: 140, B: 220, A: 255}
		if student.Status == agents.Serving || student.Status == agents.Paying {
			fill = color.RGBA{R: 225, G: 140, B: 45, A: 255}
		}
		vector.FillCircle(screen, float32(student.Position.X), float32(student.Position.Y), 6, fill, true)

		isWalkingToExit := student.Status == agents.Walking && student.Target == agents.TargetExit
		if student.Meal.Main != nil && !isWalkingToExit {
			vector.FillCircle(screen, float32(student.Position.X)+8, float32(student.Position.Y), 3,
				color.RGBA{R: 90, G: 190, B: 90, A: 255}, true)
		}
	}

	if g.showStats {
		client.DrawStats(screen, world)
	}

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

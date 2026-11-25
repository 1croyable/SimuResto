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
	SettingsPage  *client.SettingsPage
	pausedSpeed   float64
}

func (g *Game) Update() error {
	// process mouse button
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	x, y := ebiten.CursorPosition()

	// Toggle statistics overlay
	if (g.Mode == client.Running || g.Mode == client.Paused) && inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.showStats = !g.showStats
	}

	switch g.Mode {
	case client.Home:
		if clicked {
			if client.MenuButtonHit(x, y, client.Button1Y) {
				g.SettingsPage = client.NewSettingsPage(&g.Settings)
				g.Mode = client.Configuring
			} else if client.MenuButtonHit(x, y, client.Button2Y) {
				return ebiten.Termination
			}
		}
		return nil

	case client.Configuring:
		saved, canceled := g.SettingsPage.Update()
		if canceled {
			g.SettingsPage = nil
			g.Mode = client.Home
			return nil
		}
		if saved {
			g.SettingsPage = nil
			g.startSimulation()
		}
		return nil

	case client.Paused:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.resumeSimulation()
			return nil
		}
		if clicked {
			switch {
			case client.PauseDecreaseHit(x, y):
				g.pausedSpeed = client.AdjustPauseSpeed(g.pausedSpeed, -1)
			case client.PauseIncreaseHit(x, y):
				g.pausedSpeed = client.AdjustPauseSpeed(g.pausedSpeed, +1)
			case client.MenuButtonHit(x, y, client.Button1Y+65):
				g.resumeSimulation()
			case client.MenuButtonHit(x, y, client.Button2Y+65):
				g.startSimulation()
			case client.MenuButtonHit(x, y, client.Button3Y+65):
				g.closeSimulation()
				g.Mode = client.Home
			}
		}
		return nil
	}

	// pause
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.pausedSpeed = g.Simulation.World.SimulationSpeed
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
	g.frameLimiter.Wait(max(1, g.Settings.FramesPerSecond))

	screen.Fill(color.White)

	if g.Mode == client.Home {
		client.DrawMenu(screen, client.Home)
		return
	}

	if g.Mode == client.Configuring {
		client.DrawMenu(screen, client.Home)
		g.SettingsPage.Draw(screen)
		return
	}

	// Read the cached snapshot from Engine.
	snapshot := g.Simulation.Snapshot()
	client.DrawFloorplan(screen, snapshot)

	// Students
	for _, student := range snapshot.Students {
		var fill color.RGBA
		switch student.Status {
		case agents.Finished, agents.Scheduled:
			continue

		case agents.Abandoned:
			fill = color.RGBA{R: 140, G: 140, B: 140, A: 255}

			if !student.IsLeaving {
				continue
			}

		case agents.Serving, agents.Paying:
			fill = color.RGBA{R: 225, G: 140, B: 45, A: 255}

		default:
			fill = student.Student.Companion.Color
			if fill.A == 0 {
				fill = color.RGBA{R: 65, G: 140, B: 220, A: 255}
			}
		}

		x, y := float32(student.Position.X), float32(student.Position.Y)
		vector.FillCircle(screen, x, y, 6, fill, true)

		// Show a green indicator when carrying a main course.
		isWalkingToExit := student.Status == agents.Walking &&
			student.Target == agents.TargetExit

		if student.Meal.Main != nil && !isWalkingToExit {
			vector.FillCircle(screen, x+8, y, 3, color.RGBA{R: 90, G: 190, B: 90, A: 255}, true)
		}
	}

	// F3 statistics overlay
	if g.showStats {
		client.DrawStats(screen, snapshot)
	}

	// Pause menu overlay
	if g.Mode == client.Paused {
		client.DrawPauseMenu(screen, g.pausedSpeed)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return environment.Width, environment.Height
}

func (g *Game) resumeSimulation() {
	if g.Simulation != nil {
		g.Simulation.SetSimulationSpeed(g.pausedSpeed, g.Simulation.World)
	}
	g.Mode = client.Running
}

func (g *Game) startSimulation() {
	g.closeSimulation()

	ebiten.SetTPS(g.Settings.TicksPerSecond)
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

// Client is a minimal, independent view of the cafeteria.
package main

import (
	"image/color"
	"log"
	"time"

	agents "simuresto/SimuResto/Agents"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	World         *environment.Environment
	lastFrame     time.Time
	tickRemainder int64
}

func (g *Game) Update() error {
	// pause
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.World.Paused = !g.World.Paused
	}

	// restart
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.World = environment.New(g.World.Settings)
		g.tickRemainder = 0
		return nil
	}

	g.tickRemainder += int64(time.Second)
	rate := int64(g.World.Settings.TicksPerSecond)
	dt := time.Duration(g.tickRemainder / rate)
	g.tickRemainder %= rate

	// update
	engine.Step(g.World, dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Frame pacing belongs to the client; the engine only receives simulated time.
	interval := time.Second / time.Duration(g.World.Settings.FramesPerSecond)
	if delay := interval - time.Since(g.lastFrame); delay > 0 {
		time.Sleep(delay)
	}
	g.lastFrame = time.Now()

	screen.Fill(color.White)
	roadColor := color.RGBA{R: 96, G: 96, B: 96, A: 255}
	foodColor := color.RGBA{R: 224, G: 224, B: 224, A: 255}

	vector.FillRect(screen, g.World.FoodRegion.X, g.World.FoodRegion.Y, g.World.FoodRegion.Width, g.World.FoodRegion.Height, foodColor, false)
	vector.FillRect(screen, g.World.CashierRegion.X, g.World.CashierRegion.Y, g.World.CashierRegion.Width, g.World.CashierRegion.Height, foodColor, false)
	vector.FillRect(screen, g.World.FoodRegion.X, g.World.FoodRegion.Y, g.World.FoodRegion.Width, g.World.FoodRegion.Height, foodColor, false)

	for _, area := range g.World.DiningAreas {
		vector.FillRect(screen, area.Region.X, area.Region.Y, area.Region.Width, area.Region.Height, color.RGBA{R: 224, G: 234, B: 228, A: 255}, false)
	}

	vector.FillRect(screen, g.World.WasteRegion.X, g.World.WasteRegion.Y, g.World.WasteRegion.Width, g.World.WasteRegion.Height, color.RGBA{R: 237, G: 225, B: 210, A: 255}, false)

	for _, road := range g.World.Roads {
		vector.FillRect(screen, road.X, road.Y, road.Width, road.Height, roadColor, false)
	}

	ebitenutil.DebugPrintAt(screen, "Exit >", environment.Width-58, 512)

	cook := g.World.Cook.Position
	vector.FillCircle(screen, float32(cook.X), float32(cook.Y), 15,
		color.RGBA{R: 225, G: 140, B: 45, A: 255}, true)
	cashierColor := color.RGBA{R: 120, G: 120, B: 120, A: 255}
	if g.World.Cashier.StudentID != 0 {
		cashierColor = color.RGBA{R: 225, G: 140, B: 45, A: 255}
	}

	register := g.World.Cashier.Position
	vector.FillCircle(screen, float32(register.X), float32(register.Y), 15, cashierColor, true)

	finished, abandoned := 0, 0
	for _, student := range g.World.Students {
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
		if student.Meal.Main != nil && student.Status != agents.WalkingToExit {
			vector.FillCircle(screen, float32(student.Position.X)+8, float32(student.Position.Y), 3,
				color.RGBA{R: 90, G: 190, B: 90, A: 255}, true)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return environment.Width, environment.Height
}

func main() {
	settings := environment.DefaultSettings()
	world := environment.New(settings)
	ebiten.SetWindowSize(environment.Width, environment.Height)
	ebiten.SetWindowTitle("SimuResto")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(settings.TicksPerSecond)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(&Game{World: world}); err != nil {
		log.Fatal(err)
	}
}

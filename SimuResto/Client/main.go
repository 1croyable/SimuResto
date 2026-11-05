// Client is a minimal, independent view of the cafeteria.
package main

import (
	"image/color"
	"log"
	"time"

	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
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

	// 60 ticks equal one second. Prepare the simulated time to advance the world by.
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

	// Food region sits below the lower road.
	food := g.World.FoodRegion
	vector.DrawFilledRect(screen, food.X, food.Y, food.Width, food.Height, foodColor, false)

	// Route: upper-right entrance, left turn, then right toward the food region.
	for _, road := range g.World.Roads {
		vector.DrawFilledRect(screen, road.X, road.Y, road.Width, road.Height, roadColor, false)
	}
	cook := g.World.Cook.Position
	vector.DrawFilledCircle(screen, float32(cook.X), float32(cook.Y), 15,
		color.RGBA{R: 225, G: 140, B: 45, A: 255}, true)
	student := g.World.Student
	if g.World.TimePassed >= student.Student.ArrivalTime {
		vector.DrawFilledCircle(screen, float32(student.Position.X), float32(student.Position.Y), 12,
			color.RGBA{R: 65, G: 140, B: 220, A: 255}, true)
		if student.Meal.Main != nil {
			vector.DrawFilledCircle(screen, float32(student.Position.X)+17, float32(student.Position.Y), 6,
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
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	if err := ebiten.RunGame(&Game{World: world}); err != nil {
		log.Fatal(err)
	}
}

// Client is a minimal, independent view of the cafeteria.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 1000
	screenHeight = 700
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	roadColor := color.RGBA{R: 96, G: 96, B: 96, A: 255}
	foodColor := color.RGBA{R: 224, G: 224, B: 224, A: 255}

	// Food region sits below the lower road.
	vector.DrawFilledRect(screen, 155, 550, 555, 150, foodColor, false)

	// Route: upper-right entrance, left turn, then right toward the food region.
	vector.DrawFilledRect(screen, 35, 110, 965, 60, roadColor, false)
	vector.DrawFilledRect(screen, 35, 110, 60, 440, roadColor, false)
	vector.DrawFilledRect(screen, 35, 490, 845, 60, roadColor, false)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("SimuResto")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

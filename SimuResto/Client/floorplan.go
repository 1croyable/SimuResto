package client

import (
	"image/color"
	"strconv"

	agents "simuresto/SimuResto/Agents"
	engine "simuresto/SimuResto/Engine"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	tableColor = color.RGBA{R: 207, G: 224, B: 214, A: 255}
	seatColor  = color.RGBA{R: 112, G: 151, B: 130, A: 255}
)

// DrawFloorplan draws the furniture and counters over the existing simulation layout.
func DrawFloorplan(screen *ebiten.Image, snapshot engine.Snapshot) {
	layout := snapshot.Layout
	roadColor := color.RGBA{R: 96, G: 96, B: 96, A: 255}
	foodColor := color.RGBA{R: 224, G: 224, B: 224, A: 255}
	starterColor := color.RGBA{R: 180, G: 220, B: 180, A: 255}

	for _, road := range layout.Roads {
		vector.FillRect(screen, road.X, road.Y, road.Width, road.Height, roadColor, false)
	}

	for _, region := range layout.StarterRegions {
		vector.FillRect(screen, region.X, region.Y, region.Width, region.Height, starterColor, false)
	}
	main := layout.MainRegion
	vector.FillRect(screen, main.X, main.Y, main.Width, main.Height, foodColor, false)
	drawDiningFurniture(screen, layout)

	// The pickup side faces the dining aisle, across from the checkout corridor.
	station := layout.WaterAndSauceRegion
	centerX := station.X + station.Width/2
	vector.FillRect(screen, station.X, station.Y, station.Width, station.Height, color.RGBA{R: 70, G: 113, B: 140, A: 255}, false)
	portions := strconv.Itoa(snapshot.SaucePortions)
	ebitenutil.DebugPrintAt(screen, portions, int(centerX)-3*len(portions), int(station.Y+station.Height/2)-8)

	waste := layout.WasteRegion
	vector.FillRect(screen, waste.X, waste.Y, waste.Width, waste.Height, color.RGBA{R: 237, G: 225, B: 210, A: 255}, false)
	if len(layout.ExitRoute) > 0 {
		exit := layout.ExitRoute[len(layout.ExitRoute)-1]
		ebitenutil.DebugPrintAt(screen, "Exit >", int(exit.X)-58, int(exit.Y)-8)
	}

	for _, worker := range snapshot.Staff {
		fill := color.RGBA{R: 225, G: 140, B: 45, A: 255}
		x, y := float32(worker.Position.X), float32(worker.Position.Y)
		if worker.Staff.Role == agents.Cashier {
			register := layout.Cashier1Region
			if y < register.Y {
				register = layout.Cashier2Region
			}
			vector.FillRect(screen, register.X, register.Y, register.Width, register.Height, foodColor, false)
			if worker.StudentID == 0 {
				fill = color.RGBA{R: 120, G: 120, B: 120, A: 255}
			}
		}
		vector.FillCircle(screen, x, y, 15, fill, true)
	}
}

func drawDiningFurniture(screen *ebiten.Image, layout engine.LayoutSnapshot) {
	for _, tabletop := range layout.Tabletops {
		vector.FillRect(screen, tabletop.X, tabletop.Y, tabletop.Width, tabletop.Height, tableColor, false)
	}
	for _, seat := range layout.SeatPositions {
		x, y := float32(seat.X), float32(seat.Y)
		const size = engine.SeatSize
		vector.FillRect(screen, x-size/2, y-size/2, size, size, seatColor, false)
	}
}

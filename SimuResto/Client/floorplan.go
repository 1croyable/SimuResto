package client

import (
	"image/color"
	"strconv"

	agents "simuresto/SimuResto/Agents"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	tableWidth = 28
	seatSize   = 8
	seatOffset = tableWidth/2 + 2 + seatSize/2
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
	window := layout.DiningAreas[0].Region
	const windowSeats = 30
	vector.FillRect(screen, window.X, window.Y, window.Width, 30, tableColor, false)
	for i := range windowSeats {
		x := window.X + (float32(i)+0.5)*window.Width/windowSeats
		drawSeat(screen, x, window.Y+41)
	}

	left := layout.DiningAreas[1].Region
	// Keep a cross aisle between the table ends and the sauce queue.
	longTableHeight := layout.WaterAndSauceRegion.Y - 48 - left.Y
	seatsPerSide := [...]int{7, 7, 7, 7, 6}
	leftColumnWidth := left.Width / float32(1+len(seatsPerSide))
	firstX := left.X + leftColumnWidth/2
	drawSquareTableColumn(screen, left, firstX)
	for i, seats := range seatsPerSide {
		x := firstX + float32(i+1)*leftColumnWidth
		drawVerticalTable(screen, x, left.Y, longTableHeight, seats)
	}

	right := layout.DiningAreas[2].Region
	const columns, rows = 4, 4
	for row := range rows {
		y := right.Y + (float32(row)+0.5)*right.Height/rows
		for col := range columns {
			x := right.X + (float32(col)+0.5)*right.Width/columns
			drawPairedTable(screen, x, y)
		}
	}
}

func drawSquareTableColumn(screen *ebiten.Image, area environment.Region, x float32) {
	const rows = 4
	rowHeight := area.Height / rows
	for row := range rows {
		y := area.Y + (float32(row)+0.5)*rowHeight
		vector.FillRect(screen, x-tableWidth/2, y-tableWidth/2, tableWidth, tableWidth, tableColor, false)
		drawSeat(screen, x-seatOffset, y)
		drawSeat(screen, x+seatOffset, y)
		drawSeat(screen, x, y-seatOffset)
		drawSeat(screen, x, y+seatOffset)
	}
}

func drawPairedTable(screen *ebiten.Image, x, y float32) {
	const seam = 1
	for half := range 2 {
		centerX := x + (float32(half)-0.5)*(tableWidth+seam)
		vector.FillRect(screen, centerX-tableWidth/2, y-tableWidth/2, tableWidth, tableWidth, tableColor, false)
		drawSeat(screen, centerX, y-seatOffset)
		drawSeat(screen, centerX, y+seatOffset)
	}
}

func drawVerticalTable(screen *ebiten.Image, x, top, height float32, seatsPerSide int) {
	seatSpacing := height / float32(seatsPerSide)
	vector.FillRect(screen, x-tableWidth/2, top, tableWidth, height, tableColor, false)
	for i := range seatsPerSide {
		y := top + (float32(i)+0.5)*seatSpacing
		drawSeat(screen, x-seatOffset, y)
		drawSeat(screen, x+seatOffset, y)
	}
}

func drawSeat(screen *ebiten.Image, x, y float32) {
	vector.FillRect(screen, x-seatSize/2, y-seatSize/2, seatSize, seatSize, seatColor, false)
}

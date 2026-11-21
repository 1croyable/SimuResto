package environment

import (
	"slices"

	agents "simuresto/SimuResto/Agents"
)

const (
	tableWidth = 28
	SeatSize   = 8
	seatOffset = tableWidth/2 + 2 + SeatSize/2
)

type DiningSeat struct {
	Position  agents.Point
	Table     int
	Neighbors []int
	Node      int
	Access    agents.Point
}

type DiningLayout struct {
	Tabletops []Region
	Seats     []DiningSeat

	tableCount int
	obstacles  []Region
	nodes      []agents.Point
	edges      [][]int
	waiting    []int

	CheckoutNodes [MaxCashiers]int
	SauceEntrance int
	SauceExit     int
	TrayReturn    int
}

func newDiningLayout(world *Environment, areas [3]Region, aisleX float64, tray agents.Point) DiningLayout {
	var layout DiningLayout
	sauce := world.WaterAndSauceRegion
	window, left, right := areas[0], areas[1], areas[2]
	type seatEntry struct {
		index     int
		direction agents.Point
	}
	var entries []seatEntry
	addTable := func(bounds Region, parts ...Region) int {
		layout.obstacles = append(layout.obstacles, bounds)
		if len(parts) == 0 {
			parts = []Region{bounds}
		}
		layout.Tabletops = append(layout.Tabletops, parts...)
		table := layout.tableCount
		layout.tableCount++
		return table
	}
	addSeat := func(table int, x, y, dx, dy float64) int {
		index := len(layout.Seats)
		entries = append(entries, seatEntry{index, agents.Point{X: dx, Y: dy}})
		layout.Seats = append(layout.Seats, DiningSeat{Table: table, Position: agents.Point{X: x, Y: y}})
		return index
	}
	joinSeats := func(first, second int) {
		layout.Seats[first].Neighbors = append(layout.Seats[first].Neighbors, second)
		layout.Seats[second].Neighbors = append(layout.Seats[second].Neighbors, first)
	}

	table := addTable(Region{window.X, window.Y, window.Width, 30})
	const windowSeats = 30
	for i := range windowSeats {
		x := float64(window.X + (float32(i)+0.5)*window.Width/windowSeats)
		index := addSeat(table, x, float64(window.Y)+41, 0, 1)
		if i > 0 {
			joinSeats(index-1, index)
		}
	}

	const rows, rightColumns = 4, 4
	leftCounts := [...]int{7, 7, 7, 7, 6}
	leftStride := left.Width / float32(len(leftCounts)+1)
	firstX := left.X + leftStride/2
	// The square tables have a seat on each side, with adjacent seats around their perimeter.
	for row := range rows {
		y := left.Y + (float32(row)+0.5)*left.Height/rows
		table := addTable(Region{firstX - tableWidth/2, y - tableWidth/2, tableWidth, tableWidth})
		x, cy := float64(firstX), float64(y)
		indices := [4]int{
			addSeat(table, x-seatOffset, cy, -1, 0),
			addSeat(table, x, cy-seatOffset, 0, -1),
			addSeat(table, x+seatOffset, cy, 1, 0),
			addSeat(table, x, cy+seatOffset, 0, 1),
		}
		for i, index := range indices {
			joinSeats(index, indices[(i+1)%len(indices)])
		}
	}

	longHeight := sauce.Y - 48 - left.Y
	for col, count := range leftCounts {
		x := firstX + float32(col+1)*leftStride
		table := addTable(Region{x - tableWidth/2, left.Y, tableWidth, longHeight})
		for _, side := range [...]float64{-1, 1} {
			for row := range count {
				y := float64(left.Y + (float32(row)+0.5)*longHeight/float32(count))
				index := addSeat(table, float64(x)+side*seatOffset, y, side, 0)
				if row > 0 {
					joinSeats(index-1, index)
				}
			}
		}
	}

	const seam = 1
	rightStride := right.Width / rightColumns
	for row := range rows {
		y := right.Y + (float32(row)+0.5)*right.Height/rows
		for col := range rightColumns {
			x := right.X + (float32(col)+0.5)*rightStride
			bounds := Region{x - tableWidth - seam/2.0, y - tableWidth/2, 2*tableWidth + seam, tableWidth}
			table := addTable(bounds, Region{bounds.X, bounds.Y, tableWidth, tableWidth},
				Region{bounds.X + tableWidth + seam, bounds.Y, tableWidth, tableWidth})
			for _, side := range [...]float64{-1, 1} {
				seatY := float64(y) + side*seatOffset
				left := addSeat(table, float64(x)-(tableWidth+seam)/2.0, seatY, 0, side)
				right := addSeat(table, float64(x)+(tableWidth+seam)/2.0, seatY, 0, side)
				joinSeats(left, right)
			}
		}
	}

	layout.obstacles = append(layout.obstacles, sauce, world.Cashier2Region, world.CashierRegion)
	xs := make([]float64, 0, len(leftCounts)+rightColumns+3)
	for i := range len(leftCounts) + 1 {
		xs = append(xs, float64(left.X+float32(i)*leftStride))
	}
	xs = append(xs, aisleX)
	leftColumns := len(xs)
	for i := range rightColumns + 1 {
		xs = append(xs, float64(right.X+float32(i)*rightStride))
	}
	top := float64(window.Y + window.Height + 5)
	bottom := float64(right.Y + right.Height)
	ys := []float64{top}
	for i := 1; i < rows; i++ {
		ys = append(ys, float64(right.Y+float32(i)*right.Height/rows))
	}
	queueY := float64(sauce.Y) - 12
	grid := make(map[agents.Point]int)
	layout.addAisles(grid, xs[:leftColumns], append(slices.Clone(ys), queueY-16, queueY, bottom))
	layout.addAisles(grid, xs[leftColumns-1:], append(ys, bottom))
	for _, entry := range entries {
		layout.connectSeat(entry.index, entry.direction)
	}

	layout.SauceEntrance = grid[agents.Point{X: aisleX, Y: queueY}]
	layout.SauceExit = grid[agents.Point{X: aisleX, Y: queueY - 16}]
	lower := world.Checkout1Route[len(world.Checkout1Route)-1]
	upper := world.Checkout2Route[len(world.Checkout2Route)-1]
	exitY := world.ExitRoute[0].Y
	bottomLeft := grid[agents.Point{X: aisleX, Y: bottom}]
	upperTurn := layout.addNode(agents.Point{X: aisleX, Y: upper.Y})
	exitTurn := layout.addNode(agents.Point{X: aisleX, Y: exitY})
	lowerTurn := layout.addNode(agents.Point{X: aisleX, Y: lower.Y})
	wasteTurn := layout.addNode(agents.Point{X: tray.X, Y: exitY})
	layout.CheckoutNodes[0] = layout.addNode(lower)
	layout.CheckoutNodes[1] = layout.addNode(upper)
	layout.TrayReturn = layout.addNode(tray)
	layout.link(bottomLeft, upperTurn)
	layout.link(upperTurn, layout.CheckoutNodes[1])
	layout.link(upperTurn, exitTurn)
	layout.link(exitTurn, lowerTurn)
	layout.link(lowerTurn, layout.CheckoutNodes[0])
	layout.link(exitTurn, wasteTurn)
	layout.link(wasteTurn, layout.TrayReturn)

	rightEdge := float64(right.X + right.Width)
	layout.buildWaitingLoop(bottomLeft, grid[agents.Point{X: rightEdge, Y: bottom}],
		grid[agents.Point{X: rightEdge, Y: top}], grid[agents.Point{X: aisleX, Y: top}])
	return layout
}

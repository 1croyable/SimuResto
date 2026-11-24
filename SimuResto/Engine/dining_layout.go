package engine

import (
	"slices"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

const (
	tableWidth = 28
	SeatSize   = 8
	seatOffset = tableWidth/2 + 2 + SeatSize/2
)

type DiningSeat struct {
	Position  agents.Point
	Table     int
	Neighbors []int // Neighbors contains eat indexes. At a table already in use, seat selection prefers seats whose neighbors are all free.
	Node      int
	Access    agents.Point // Access is the turning point between the aisle node and the seat, set by connectSeat.
}

type DiningLayout struct {
	Tabletops []environment.Region
	Seats     []DiningSeat

	tableCount int
	obstacles  []environment.Region
	nodes      []agents.Point
	edges      [][]int
	waiting    []int

	CheckoutNodes [environment.MaxCashiers]int
	SauceEntrance int
	SauceExit     int
	TrayReturn    int
}

func newDiningLayout(world *environment.Environment) DiningLayout {
	var layout DiningLayout
	sauce := world.WaterAndSauceRegion
	const aisleX, diningTop = 770, 200
	laneTop := world.Cashier2Region.Y + world.Cashier2Region.Height
	diningHeight := laneTop - 25 - diningTop
	window := environment.Region{X: 180, Y: 120, Width: 820, Height: 55}
	left := environment.Region{X: 130, Y: diningTop, Width: 605, Height: diningHeight}
	right := environment.Region{X: 805, Y: diningTop, Width: 335, Height: diningHeight}
	type seatEntry struct {
		index     int
		direction agents.Point
	}
	var entries []seatEntry
	addTable := func(bounds environment.Region, parts ...environment.Region) int {
		layout.obstacles = append(layout.obstacles, bounds)
		if len(parts) == 0 {
			parts = []environment.Region{bounds}
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

	table := addTable(environment.Region{X: window.X, Y: window.Y, Width: window.Width, Height: 30})
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
		table := addTable(environment.Region{
			X: firstX - tableWidth/2, Y: y - tableWidth/2, Width: tableWidth, Height: tableWidth,
		})
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
		table := addTable(environment.Region{X: x - tableWidth/2, Y: left.Y, Width: tableWidth, Height: longHeight})
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
			bounds := environment.Region{
				X: x - tableWidth - seam/2.0, Y: y - tableWidth/2, Width: 2*tableWidth + seam, Height: tableWidth,
			}
			first := environment.Region{X: bounds.X, Y: bounds.Y, Width: tableWidth, Height: tableWidth}
			second := first
			second.X += tableWidth + seam
			table := addTable(bounds, first, second)
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
	exit := world.ExitRoute[0]
	tray := agents.Point{X: exit.X, Y: float64(world.WasteRegion.Y + world.WasteRegion.Height/2)}
	bottomLeft := grid[agents.Point{X: aisleX, Y: bottom}]
	upperTurn := layout.addNode(agents.Point{X: aisleX, Y: upper.Y})
	exitTurn := layout.addNode(agents.Point{X: aisleX, Y: exit.Y})
	lowerTurn := layout.addNode(agents.Point{X: aisleX, Y: lower.Y})
	wasteTurn := layout.addNode(exit)
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

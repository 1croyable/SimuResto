package environment

import (
	"fmt"
	"math"
	"slices"

	agents "simuresto/SimuResto/Agents"
)

func (layout *DiningLayout) addNode(point agents.Point) int {
	index := len(layout.nodes)
	layout.nodes = append(layout.nodes, point)
	layout.edges = append(layout.edges, nil)
	return index
}

func (layout *DiningLayout) link(a, b int) {
	if !layout.clear(layout.nodes[a], layout.nodes[b]) {
		panic("dining path crosses a counter or tabletop")
	}
	layout.edges[a] = append(layout.edges[a], b)
	layout.edges[b] = append(layout.edges[b], a)
}

// All paths are orthogonal. Leave enough clearance for the student circle.
func (layout *DiningLayout) clear(a, b agents.Point) bool {
	return layout.clearWithMargin(a, b, 8)
}

func (layout *DiningLayout) clearWithMargin(a, b agents.Point, clearance float64) bool {
	for _, obstacle := range layout.obstacles {
		if max(a.X, b.X) > float64(obstacle.X)-clearance &&
			min(a.X, b.X) < float64(obstacle.X+obstacle.Width)+clearance &&
			max(a.Y, b.Y) > float64(obstacle.Y)-clearance &&
			min(a.Y, b.Y) < float64(obstacle.Y+obstacle.Height)+clearance {
			return false
		}
	}
	return true
}

func (layout *DiningLayout) addAisles(grid map[agents.Point]int, xs, ys []float64) {
	for row, y := range ys {
		for col, x := range xs {
			point := agents.Point{X: x, Y: y}
			if !layout.clear(point, point) {
				continue
			}
			node, exists := grid[point]
			if !exists {
				node = layout.addNode(point)
				grid[point] = node
			}
			if col > 0 {
				previous := agents.Point{X: xs[col-1], Y: y}
				if other, ok := grid[previous]; ok && layout.clear(previous, point) {
					layout.link(other, node)
				}
			}
			if row > 0 && !exists {
				previous := agents.Point{X: x, Y: ys[row-1]}
				if other, ok := grid[previous]; ok && layout.clear(previous, point) {
					layout.link(other, node)
				}
			}
		}
	}
}

func (layout *DiningLayout) connectSeat(index int, direction agents.Point) {
	seat := &layout.Seats[index]
	seat.Node = -1
	best := math.Inf(1)
	for node, point := range layout.nodes {
		dx, dy := point.X-seat.Position.X, point.Y-seat.Position.Y
		if dx*direction.X+dy*direction.Y <= 0 {
			continue
		}
		access := agents.Point{X: point.X, Y: seat.Position.Y}
		if direction.Y != 0 {
			access = agents.Point{X: seat.Position.X, Y: point.Y}
		}
		distance := math.Abs(dx) + math.Abs(dy)
		// The student's circle can touch the tabletop edge at the seat.
		if distance < best && layout.clear(point, access) && layout.clearWithMargin(access, seat.Position, 6) {
			seat.Node = node
			seat.Access = access
			best = distance
		}
	}
	if seat.Node == -1 {
		panic(fmt.Sprintf("seat %d at %v has no access from a dining aisle", index+1, seat.Position))
	}
}

type DiningSearch struct {
	Distance []int // Number of edges from the search origin, or -1 if unreachable.
	previous []int
}

func (layout *DiningLayout) Search(start int) DiningSearch {
	search := DiningSearch{Distance: make([]int, len(layout.nodes)), previous: make([]int, len(layout.nodes))}
	for i := range search.Distance {
		search.Distance[i], search.previous[i] = -1, -1
	}
	queue := []int{start}
	search.Distance[start] = 0
	for i := 0; i < len(queue); i++ {
		from := queue[i]
		for _, next := range layout.edges[from] {
			if search.Distance[next] == -1 {
				search.Distance[next] = search.Distance[from] + 1
				search.previous[next] = from
				queue = append(queue, next)
			}
		}
	}
	return search
}

// AppendRoute adds the graph waypoints after the search origin.
func (layout *DiningLayout) AppendRoute(route []agents.Point, search DiningSearch, end int) []agents.Point {
	start := len(route)
	for node := end; search.previous[node] != -1; node = search.previous[node] {
		route = append(route, layout.nodes[node])
	}
	slices.Reverse(route[start:])
	return route
}

func (layout *DiningLayout) Point(node int) agents.Point {
	return layout.nodes[node]
}

func (layout *DiningLayout) buildWaitingLoop(corners ...int) {
	// The search tree leads arrivals onto the loop. Its edges are replaced on the loop itself.
	layout.waiting = layout.Search(corners[0]).previous
	for i, from := range corners {
		to := corners[(i+1)%len(corners)]
		search := layout.Search(from)
		if search.Distance[to] < 0 {
			panic("dining waiting loop is disconnected")
		}
		for node := to; node != from; node = search.previous[node] {
			layout.waiting[search.previous[node]] = node
		}
	}
}

func (layout *DiningLayout) NextWaitingNode(node int) int {
	return layout.waiting[node]
}

package main

import (
	"fmt"
	"image/color"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

var statsFont = text.NewGoXFace(basicfont.Face7x13)

func drawStatsText(screen *ebiten.Image, label string, x, y int) {
	draw := func(dx, dy float64, clr color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Scale(1.2, 1.2)
		op.GeoM.Translate(float64(x)+dx, float64(y)+dy)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, label, statsFont, op)
	}

	draw(-1, 0, color.Black)
	draw(1, 0, color.Black)
	draw(0, -1, color.Black)
	draw(0, 1, color.Black)
	draw(0, 0, color.White)
}

func drawStatsPanel(screen *ebiten.Image, lines []string, x, y, width int) {
	const padding = 12
	const lineHeight = 20

	height := len(lines)*lineHeight + padding*2

	vector.FillRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(height),
		color.RGBA{R: 40, G: 40, B: 40, A: 190},
		false,
	)

	for i, line := range lines {
		drawStatsText(screen, line, x+padding, y+padding+i*lineHeight)
	}
}

// Calculate the average waiting time at a counter.
func averageWait(stats environment.CounterStatistics) float64 {
	if stats.StudentsServed == 0 {
		return 0
	}
	return stats.TotalWaitTime.Seconds() / float64(stats.StudentsServed)
}

// Calculate staff utilization as a percentage.
func utilization(stats environment.CounterStatistics, elapsed float64) float64 {
	if elapsed <= 0 {
		return 0
	}
	return stats.TotalServiceTime.Seconds() / elapsed * 100
}

func drawStats(screen *ebiten.Image, world *environment.Environment) {
	// Student statistics
	active, finished, abandoned := 0, 0, 0

	for _, student := range world.Students {
		switch student.Status {
		case agents.Finished:
			finished++
		case agents.Abandoned:
			abandoned++
		case agents.Scheduled:
			continue
		default:
			active++
		}
	}

	queueLength := 0
	if world.Queue != nil {
		queueLength = len(world.Queue.IDs())
	}

	elapsed := world.TimePassed.Seconds()
	seconds := int(elapsed)
	timeLabel := fmt.Sprintf("%02d:%02d:%02d",
		seconds/3600, (seconds/60)%60, seconds%60)

	stats := &world.Statistics

	// Left panel: global simulation statistics
	leftLines := []string{
		"SIMURESTO - SIMULATION",
		"",
		fmt.Sprintf("Simulation time : %s", timeLabel),
		fmt.Sprintf("Total students  : %d", len(world.Students)),
		fmt.Sprintf("Active students : %d", active),
		fmt.Sprintf("Queue length    : %d", queueLength),
		fmt.Sprintf("Maximum queue   : %d", stats.MaxQueueLength),
		fmt.Sprintf("Finished        : %d", finished),
		fmt.Sprintf("Abandoned       : %d", abandoned),
		"",
		fmt.Sprintf("FPS / TPS       : %.1f / %.1f",
			ebiten.ActualFPS(), ebiten.ActualTPS()),
	}

	// Right panel: operational statistics
	rightLines := []string{
		"SIMURESTO - OPERATIONS",
		"",
		"FOOD SERVICE",
		fmt.Sprintf("Starter 1 : %d students / %d dishes",
			stats.StarterCounters[0].StudentsServed,
			stats.StarterCounters[0].DishesCollected),
		fmt.Sprintf("Starter 2 : %d students / %d dishes",
			stats.StarterCounters[1].StudentsServed,
			stats.StarterCounters[1].DishesCollected),
		fmt.Sprintf("Main      : %d dishes",
			stats.MainCounter.DishesCollected),
		fmt.Sprintf("Cashier   : %d students",
			stats.CashierCounter.StudentsServed),
		"",
		"AVERAGE WAITING TIME",
		fmt.Sprintf("Starter 1 : %.2f s", averageWait(stats.StarterCounters[0])),
		fmt.Sprintf("Starter 2 : %.2f s", averageWait(stats.StarterCounters[1])),
		fmt.Sprintf("Main      : %.2f s", averageWait(stats.MainCounter)),
		fmt.Sprintf("Cashier   : %.2f s", averageWait(stats.CashierCounter)),
		"",
		"STAFF UTILIZATION",
		fmt.Sprintf("Cook      : %.1f%%", utilization(stats.MainCounter, elapsed)),
		fmt.Sprintf("Cashier   : %.1f%%", utilization(stats.CashierCounter, elapsed)),
		"",
		"AVAILABLE FOOD STOCK",
	}

	for _, dish := range world.Menu {
		rightLines = append(rightLines,
			fmt.Sprintf("%-18s : %d", dish.Name, dish.Portions))
	}

	const panelWidth = 430
	const margin = 10

	drawStatsPanel(screen, leftLines, margin, margin, panelWidth)
	drawStatsPanel(screen, rightLines, environment.Width-panelWidth-margin, margin, panelWidth)
}

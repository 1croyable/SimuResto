package client

import (
	"fmt"
	"image/color"
	"time"

	agents "simuresto/SimuResto/Agents"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

var statsFont = text.NewGoXFace(basicfont.Face7x13)

func DrawStatsText(screen *ebiten.Image, label string, x, y int) {
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

func DrawStatsPanel(screen *ebiten.Image, lines []string, x, y, width int) {
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
		DrawStatsText(screen, line, x+padding, y+padding+i*lineHeight)
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

func DrawStats(screen *ebiten.Image, snapshot engine.Snapshot) {
	metrics := snapshot.Metrics
	stats := snapshot.Statistics

	// Simulation time.
	seconds := int(snapshot.Duration / time.Second)
	timeLabel := fmt.Sprintf(
		"%02d:%02d:%02d",
		seconds/3600,
		(seconds/60)%60,
		seconds%60,
	)

	// clock (11:30 at simulation start).
	clock := snapshot.Settings.StartTime + snapshot.Duration

	clockSeconds := int(clock / time.Second)
	clockLabel := fmt.Sprintf(
		"%02d:%02d:%02d",
		(clockSeconds/3600)%24,
		(clockSeconds/60)%60,
		clockSeconds%60,
	)

	// Left panel: global simulation statistics.
	leftLines := []string{
		"SIMURESTO - SIMULATION",
		"",
		fmt.Sprintf("Restaurant time : %s", clockLabel),
		fmt.Sprintf("Operating phase : %s", snapshot.OperatingPhase),
		fmt.Sprintf("Elapsed time    : %s", timeLabel),
		fmt.Sprintf("Total students  : %d", metrics.TotalStudents),
		fmt.Sprintf("Scheduled       : %d", metrics.Scheduled),
		fmt.Sprintf("Admitted        : %d", metrics.Admitted),
		fmt.Sprintf("Active students : %d", metrics.Active),
		fmt.Sprintf("Finished        : %d", metrics.Completed),
		fmt.Sprintf("Abandoned       : %d", metrics.Abandoned),
		"",
		"QUEUE",
		fmt.Sprintf(
			"Current / Capacity : %d / %d",
			metrics.QueueLength,
			metrics.QueueCapacity,
		),
		fmt.Sprintf("Maximum queue   : %d", metrics.MaxQueueLength),
		fmt.Sprintf("Abandonment rate: %.1f%%", metrics.AbandonmentRate*100),
		"",
		"THROUGHPUT",
		fmt.Sprintf("Payments done   : %d", metrics.MealsServed),
		fmt.Sprintf("Meals / minute  : %.2f", metrics.MealsPerMinute),
		fmt.Sprintf("Avg service wait: %.2f s",
			metrics.AverageServiceWait.Seconds()),
		"",
		fmt.Sprintf(
			"FPS / TPS       : %.1f / %.1f",
			ebiten.ActualFPS(),
			ebiten.ActualTPS(),
		),
	}

	// Right panel: operational statistics.
	rightLines := []string{
		"SIMURESTO - OPERATIONS",
		"",
		"FOOD SERVICE",
		fmt.Sprintf(
			"Starter 1 : %d students / %d dishes",
			stats.StarterCounters[0].StudentsServed,
			stats.StarterCounters[0].DishesCollected,
		),
		fmt.Sprintf(
			"Starter 2 : %d students / %d dishes",
			stats.StarterCounters[1].StudentsServed,
			stats.StarterCounters[1].DishesCollected,
		),
		fmt.Sprintf(
			"Main      : %d dishes",
			stats.MainCounter.DishesCollected,
		),
		fmt.Sprintf(
			"Cashier   : %d students",
			stats.CashierCounter.StudentsServed,
		),
		"",
		"AVERAGE WAITING TIME",
		fmt.Sprintf(
			"Starter 1 : %.2f s",
			metrics.AverageWaitByCounter["Starter 1"].Seconds(),
		),
		fmt.Sprintf(
			"Starter 2 : %.2f s",
			metrics.AverageWaitByCounter["Starter 2"].Seconds(),
		),
		fmt.Sprintf(
			"Main      : %.2f s",
			metrics.AverageWaitByCounter["Main"].Seconds(),
		),
		fmt.Sprintf(
			"Cashier   : %.2f s",
			metrics.AverageWaitByCounter["Cashier"].Seconds(),
		),
		"",
		"MAXIMUM WAITING TIME",
		fmt.Sprintf(
			"Starter 1 : %.2f s",
			metrics.MaxWaitByCounter["Starter 1"].Seconds(),
		),
		fmt.Sprintf(
			"Starter 2 : %.2f s",
			metrics.MaxWaitByCounter["Starter 2"].Seconds(),
		),
		fmt.Sprintf(
			"Main      : %.2f s",
			metrics.MaxWaitByCounter["Main"].Seconds(),
		),
		fmt.Sprintf(
			"Cashier   : %.2f s",
			metrics.MaxWaitByCounter["Cashier"].Seconds(),
		),
		"",
		"STAFF UTILIZATION",
		fmt.Sprintf(
			"Cook      : %.1f%%",
			metrics.StaffUtilization[agents.Server],
		),
		fmt.Sprintf(
			"Cashier   : %.1f%%",
			metrics.StaffUtilization[agents.Cashier],
		),
		"",
		"FOOD STOCK (AVAILABLE / RESERVED / COLLECTED)",
	}

	for _, stock := range snapshot.Stock {
		rightLines = append(rightLines,
			fmt.Sprintf(
				"%-15.15s : %d / %d / %d",
				stock.Dish.Name,
				stock.Dish.Portions,
				stock.Reserved,
				stock.Collected,
			),
		)
	}

	const panelWidth = 430
	const margin = 10

	DrawStatsPanel(screen, leftLines, margin, margin, panelWidth)
	DrawStatsPanel(
		screen,
		rightLines,
		environment.Width-panelWidth-margin,
		margin,
		panelWidth,
	)
}

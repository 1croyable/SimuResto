package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	agents "simuresto/SimuResto/Agents"
	client "simuresto/SimuResto/Client"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"
)

func main() { os.Exit(run()) }

func run() int {
	gui := flag.Bool("gui", false, "打开图形界面")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "直接运行默认模拟，或使用 -gui 打开界面")
		return 2
	}
	config := environment.DefaultConfig()
	generator := agents.StudentGenerator{Seed: environment.RandomSeed, ArrivalInterval: environment.ArrivalInterval}
	students, err := generator.Generate(environment.DefaultStudentCount)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *gui {
		err = client.Run(client.Options{Config: config, Students: students})
	} else {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		var report engine.Report
		report, err = engine.Run(ctx, config, students)
		printSummary(report)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printSummary(report engine.Report) {
	metrics := report.Metrics
	fmt.Printf("Simulated duration: %s\n", report.Duration)
	fmt.Printf("Students: %d completed, %d abandoned, %d rejected (full), %d cancelled\n",
		metrics.Completed, metrics.Abandoned, metrics.Rejected, metrics.Cancelled)
	fmt.Printf("Average wait: %s | Satisfaction: %.1f/100 | Abandonment: %.1f%% | Throughput: %.1f students/min\n\n",
		metrics.AverageWait.Round(time.Millisecond), metrics.AverageSatisfaction,
		100*metrics.AbandonmentRate, metrics.MealsPerMinute)
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "AREA\tPEAK OCCUPANCY\tAVERAGE WAIT")
	for _, area := range []agents.Area{
		agents.Outside, agents.Inside, agents.SelfService, agents.MainService, agents.Payment,
	} {
		fmt.Fprintf(writer, "%s\t%d\t%s\n", area, report.PeakOccupancy[area], metrics.AverageWaitByArea[area].Round(time.Millisecond))
	}
	fmt.Fprintln(writer, "\nDISH\tTAKEN\tREMAINING\tREPLENISHED")
	for _, stock := range report.Stock {
		fmt.Fprintf(writer, "%s\t%d\t%d\t%d\n", stock.Dish.Name, stock.Taken, stock.Remaining, stock.Replenished)
	}
	fmt.Fprintln(writer, "\nSTAFF\tSERVED\tUTILIZATION")
	for _, worker := range report.Staff {
		fmt.Fprintf(writer, "%s #%d\t%d\t%.1f%%\n", worker.Staff.Role, worker.Staff.ID, worker.Served, 100*worker.Utilization)
	}
	writer.Flush()
}

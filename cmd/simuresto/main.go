package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"github.com/1croyable/SimuResto/simulation"
)

func main() {
	os.Exit(run())
}

func run() int {
	config := simulation.DefaultConfig()
	count := flag.Int("students", 80, "number of students")
	seed := flag.Int64("seed", 42, "seed for student profiles")
	arrivalInterval := flag.Duration("arrival-interval", time.Second, "simulated time between arrivals (0 for a crowd)")
	jsonOutput := flag.Bool("json", false, "print the complete report as JSON (durations in nanoseconds)")
	events := flag.Bool("events", false, "print agent interactions before the summary")
	flag.IntVar(&config.Capacities.Outside, "outdoor-capacity", config.Capacities.Outside, "outdoor queue capacity")
	flag.IntVar(&config.Capacities.Inside, "indoor-capacity", config.Capacities.Inside, "indoor queue capacity")
	flag.IntVar(&config.Capacities.SelfService, "self-capacity", config.Capacities.SelfService, "self-service area capacity")
	flag.IntVar(&config.Capacities.MainService, "main-capacity", config.Capacities.MainService, "main-service area capacity")
	flag.IntVar(&config.Capacities.Payment, "payment-capacity", config.Capacities.Payment, "payment area capacity")
	flag.IntVar(&config.Servers, "servers", config.Servers, "main-course servers")
	flag.IntVar(&config.Cashiers, "cashiers", config.Cashiers, "cashiers")
	flag.IntVar(&config.MaxSelfServiceItems, "max-self-items", config.MaxSelfServiceItems, "maximum sides per student in total")
	flag.IntVar(&config.MaxStartersPerStudent, "max-starters", config.MaxStartersPerStudent, "maximum starters per student")
	flag.IntVar(&config.MaxDessertsPerStudent, "max-desserts", config.MaxDessertsPerStudent, "maximum desserts per student")
	flag.DurationVar(&config.Tick, "tick", config.Tick, "logical clock step")
	flag.DurationVar(&config.TickInterval, "tick-interval", config.TickInterval, "wall-clock delay per step (0 runs at full speed)")
	flag.DurationVar(&config.MaxDuration, "max-duration", config.MaxDuration, "maximum simulated duration")
	flag.DurationVar(&config.SelfServiceTime, "self-time", config.SelfServiceTime, "time to collect sides")
	flag.DurationVar(&config.MainServiceTime, "main-time", config.MainServiceTime, "time to serve a main")
	flag.DurationVar(&config.PaymentTime, "payment-time", config.PaymentTime, "time to pay")
	flag.IntVar(&config.Menu[0].Portions, "starters", config.Menu[0].Portions, "daily starter portions")
	flag.IntVar(&config.Menu[1].Portions, "vegetarian-mains", config.Menu[1].Portions, "daily vegetarian main portions")
	flag.IntVar(&config.Menu[2].Portions, "meat-mains", config.Menu[2].Portions, "daily meat main portions")
	flag.IntVar(&config.Menu[3].Portions, "desserts", config.Menu[3].Portions, "daily dessert portions")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments; use -help for options")
		return 2
	}
	students, err := simulation.GenerateStudents(*count, *seed, *arrivalInterval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	report, err := simulation.Run(ctx, config, students)
	if err != nil && report.Metrics.AverageWaitByArea == nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(report); encodeErr != nil {
			fmt.Fprintln(os.Stderr, encodeErr)
			return 1
		}
	} else {
		if *events {
			for _, event := range report.Events {
				fmt.Printf("%7s  student=%-3d staff=%-2d %-21s %s -> %s %v\n",
					event.Time, event.StudentID, event.StaffID, event.Signal, event.From, event.To, event.DishIDs)
			}
		}
		printSummary(report)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printSummary(report simulation.Report) {
	metrics := report.Metrics
	fmt.Printf("Simulated duration: %s\n", report.Duration)
	fmt.Printf("Students: %d completed, %d abandoned, %d rejected (full), %d cancelled\n",
		metrics.Completed, metrics.Abandoned, metrics.Rejected, metrics.Cancelled)
	fmt.Printf("Average wait: %s | Satisfaction: %.1f/100 | Abandonment: %.1f%% | Throughput: %.1f students/min\n\n",
		metrics.AverageWait.Round(time.Millisecond), metrics.AverageSatisfaction,
		100*metrics.AbandonmentRate, metrics.MealsPerMinute)
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "AREA\tPEAK OCCUPANCY\tAVERAGE WAIT")
	for _, area := range []simulation.Area{
		simulation.Outside, simulation.Inside, simulation.SelfService, simulation.MainService, simulation.Payment,
	} {
		fmt.Fprintf(writer, "%s\t%d\t%s\n", area, report.PeakOccupancy[area], metrics.AverageWaitByArea[area].Round(time.Millisecond))
	}
	fmt.Fprintln(writer, "\nDISH\tTAKEN\tREMAINING")
	for _, stock := range report.Stock {
		fmt.Fprintf(writer, "%s\t%d\t%d\n", stock.Dish.Name, stock.Taken, stock.Remaining)
	}
	fmt.Fprintln(writer, "\nSTAFF\tSERVED\tUTILIZATION")
	for _, worker := range report.Staff {
		fmt.Fprintf(writer, "%s #%d\t%d\t%.1f%%\n", worker.Staff.Role, worker.Staff.ID, worker.Served, 100*worker.Utilization)
	}
	writer.Flush()
}

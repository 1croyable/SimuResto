package main

import (
	"flag"
	"time"

	client "simuresto/SimuResto/Client"
	environment "simuresto/SimuResto/Environment"
)

// 图形模式与命令行共用唯一入口；窗口选项不改变模拟核心。
func graphicalFlags() *client.Options {
	options := &client.Options{}
	flag.Float64Var(&options.Speed, "speed", 4, "GUI: simulated seconds per real second")
	flag.BoolVar(&options.Paused, "paused", false, "GUI: start with simulation paused")
	flag.BoolVar(&options.AutoStart, "autostart", false, "GUI: skip setup and start immediately")
	flag.StringVar(&options.FontPath, "font", "", "GUI: optional Chinese TTF/OTF/TTC font path")
	flag.StringVar(&options.Screenshot, "screenshot", "", "GUI: save a rendered PNG and exit")
	flag.DurationVar(&options.CaptureAt, "capture-at", 60*time.Second, "GUI: simulated time to save screenshot")
	flag.DurationVar(&options.PerformanceDuration, "perf-duration", 0, "GUI: measure performance after 1s warm-up, print results and exit")
	flag.BoolVar(&options.PerformanceUncapped, "perf-uncapped", false, "GUI: temporarily disable frame cap; requires -perf-duration")
	flag.StringVar(&options.CPUProfile, "cpuprofile", "", "GUI: save a Go CPU profile")
	return options
}

// 仅调整用户没有显式指定的默认值，保留命令行模式的原有行为。
func applyGraphicalDefaults(config *environment.Config, count *int) {
	specified := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { specified[f.Name] = true })
	if !specified["students"] {
		*count = 100
	}
	if !specified["tick"] {
		config.Tick = 250 * time.Millisecond
	}
	if !specified["cashiers"] && !specified["payment-capacity"] {
		config.Cashiers, config.Capacities.Payment = 2, 2
	} else if !specified["cashiers"] {
		config.Cashiers = config.Capacities.Payment
	} else if !specified["payment-capacity"] {
		config.Capacities.Payment = config.Cashiers
	}
}

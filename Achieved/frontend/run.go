package frontend

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Run 创建窗口并运行前端，退出时关闭本轮学生和工作人员的 goroutine。
func Run(options *Options) error {
	if options.PerformanceDuration < 0 {
		return fmt.Errorf("性能测量时长不能为负数")
	}
	if options.PerformanceUncapped && options.PerformanceDuration <= 0 {
		return fmt.Errorf("-perf-uncapped 必须配合 -perf-duration，正常运行仍限制为 60 FPS")
	}
	game, err := New(*options)
	if err != nil {
		return err
	}
	defer game.Close()
	if options.CPUProfile != "" {
		file, err := os.Create(options.CPUProfile)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := pprof.StartCPUProfile(file); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}
	if options.PerformanceDuration > 0 {
		game.performance = &performanceStats{start: time.Now().Add(time.Second), duration: options.PerformanceDuration}
	}
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("SimuResto | CROUS")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(FPSLimit)
	// 不依赖显示器刷新率，由绘制限帧器保证最多 60 FPS。
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	return ebiten.RunGame(game)
}

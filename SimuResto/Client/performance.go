package client

import (
	"fmt"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// 仅在 -perf-duration 启用时采样，默认运行不收集性能数据。
// Draw 耗时只表示 CPU 提交绘制命令的时间，完整帧间隔还包含 GPU 执行和限帧等待。
type performanceStats struct {
	start                time.Time
	duration             time.Duration
	lastFrame            time.Time
	intervals            []time.Duration
	frames, updates      int
	drawTime, updateTime time.Duration
	waitTime             time.Duration
	width, height        int
}

func (p *performanceStats) measureWait(start time.Time) {
	if !start.Before(p.start) {
		p.waitTime += time.Since(start)
	}
}

func (g *Game) drawFPS(screen *ebiten.Image) {
	label := fmt.Sprintf("%.0f / %d FPS", ebiten.ActualFPS(), FPSLimit)
	if g.options.PerformanceUncapped {
		label = fmt.Sprintf(g.label("%.0f FPS 测量", "%.0f FPS uncapped"), ebiten.ActualFPS())
	}
	g.text(screen, label, 515, 33, 13, muted)
}

func (p *performanceStats) finished() bool { return time.Since(p.start) >= p.duration }

func (p *performanceStats) frame(width, height int) {
	now := time.Now()
	if now.Before(p.start) {
		return
	}
	if !p.lastFrame.IsZero() {
		p.intervals = append(p.intervals, now.Sub(p.lastFrame))
	}
	p.lastFrame = now
	p.frames++
	p.width, p.height = width, height
}

func (p *performanceStats) measureUpdate(start time.Time) {
	if start.Before(p.start) {
		return
	}
	p.updateTime += time.Since(start)
	p.updates++
}

func (p *performanceStats) measureDraw(start time.Time) {
	if !start.Before(p.start) {
		p.drawTime += time.Since(start)
	}
}

func (p *performanceStats) print() {
	slices.Sort(p.intervals)
	var p95 time.Duration
	if len(p.intervals) > 0 {
		p95 = p.intervals[(len(p.intervals)-1)*95/100]
	}
	// 使用相邻帧间隔计算帧率，避免采样窗口首帧多计一次导致约 60.1 FPS。
	var frameTime time.Duration
	for _, interval := range p.intervals {
		frameTime += interval
	}
	fps := 0.0
	if frameTime > 0 {
		fps = float64(len(p.intervals)) / frameTime.Seconds()
	}
	fmt.Printf("Performance: %dx%d | %.1f FPS | frame p95 %.2f ms | Update avg %.3f ms | Draw submit avg %.3f ms | Frame wait avg %.3f ms\n",
		p.width, p.height, fps, float64(p95)/float64(time.Millisecond),
		float64(p.updateTime)/float64(time.Millisecond)/float64(max(1, p.updates)),
		float64(p.drawTime)/float64(time.Millisecond)/float64(max(1, p.frames)),
		float64(p.waitTime)/float64(time.Millisecond)/float64(max(1, p.frames)))
}

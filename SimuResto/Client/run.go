package client

import "github.com/hajimehoshi/ebiten/v2"

// Run 创建窗口并运行前端，退出时关闭本轮学生和工作人员的 goroutine。
func Run(options Options) error {
	game, err := New(options)
	if err != nil {
		return err
	}
	defer game.Close()
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("SimuResto | CROUS")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(FPSLimit)
	// 不依赖显示器刷新率，由绘制限帧器保证最多 60 FPS。
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	return ebiten.RunGame(game)
}

package client

import (
	"math"
	"testing"
)

func TestViewportKeepsClicksAlignedAcrossDPIAndWindowSizes(t *testing.T) {
	for _, size := range [][2]float64{{1280, 800}, {1920, 1200}, {2560, 1600}, {1600, 900}, {900, 1200}} {
		v := newViewport(size[0], size[1])
		// 覆盖 100%、150%、200% DPI，以及宽窗口和高窗口的留白区域。
		for _, button := range []rectangle{pauseButton, speedButton, restartButton, cashierButton, stepButton} {
			x, y := float64(button.x+button.w/2), float64(button.y+button.h/2)
			pixelX, pixelY := v.x+x*v.scale, v.y+y*v.scale
			logical := v.canvasPoint(pixelX, pixelY)
			if math.Abs(logical.x-x) > 1e-9 || math.Abs(logical.y-y) > 1e-9 || !button.contains(int(logical.x), int(logical.y)) {
				t.Fatalf("窗口 %v 的渲染坐标与点击区域未对齐", size)
			}
		}
	}
}

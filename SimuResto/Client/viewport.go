package client

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// 平面图使用统一的设计坐标；绘制时直接转换到物理像素，而不是放大一张低分辨率图片。
type viewport struct {
	scale float64
	x, y  float64
}

func newViewport(width, height float64) viewport {
	scale := math.Min(width/Width, height/Height)
	return viewport{scale: scale, x: (width - Width*scale) / 2, y: (height - Height*scale) / 2}
}

func (v viewport) canvasPoint(x, y float64) point {
	return point{(x - v.x) / v.scale, (y - v.y) / v.scale}
}

func screenViewport(screen *ebiten.Image) viewport {
	bounds := screen.Bounds()
	return newViewport(float64(bounds.Dx()), float64(bounds.Dy()))
}

// LayoutF 结合窗口尺寸与当前屏幕 DPI 分配画布，移动到不同 DPI 的屏幕时也会重新计算。
func (g *Game) LayoutF(width, height float64) (float64, float64) {
	dpi := 1.0
	if monitor := ebiten.Monitor(); monitor != nil {
		dpi = monitor.DeviceScaleFactor()
	}
	width, height = max(1, width*dpi), max(1, height*dpi)
	g.viewport = newViewport(math.Ceil(width), math.Ceil(height))
	return width, height
}

func (g *Game) Layout(width, height int) (int, int) {
	w, h := g.LayoutF(float64(width), float64(height))
	return int(math.Ceil(w)), int(math.Ceil(h))
}

// 鼠标位置也转换回设计坐标，使高 DPI、缩放和留白下的点击区域保持一致。
func (g *Game) cursor() (int, int) {
	x, y := ebiten.CursorPosition()
	p := g.viewport.canvasPoint(float64(x), float64(y))
	return int(math.Floor(p.x)), int(math.Floor(p.y))
}

package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 静态平面图按当前物理像素缓存，窗口或参数改变时重建，显示时始终一比一拷贝。
func (g *Game) floorBackground(screen *ebiten.Image) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	if g.floorLayer == nil {
		g.floorLayer = ebiten.NewImage(width, height)
		g.floor(g.floorLayer)
	}
	screen.DrawImage(g.floorLayer, nil)
}

// 在绘制任何元素前处理尺寸变化，避免同一帧中途清理已使用的纹理。
func (g *Game) prepareRenderCache(screen *ebiten.Image) {
	size := screen.Bounds().Size()
	if g.renderSize != size {
		g.clearRenderCache()
		g.renderSize = size
	}
}

func (g *Game) clearRenderCache() {
	if g.floorLayer != nil {
		g.floorLayer.Deallocate()
		g.floorLayer = nil
	}
	for _, sprite := range g.dotSprites {
		sprite.Deallocate()
	}
	g.dotSprites = nil
	for _, sprite := range g.roundedSprites {
		sprite.image.Deallocate()
	}
	g.roundedSprites = nil
	for _, sprite := range g.circleSprites {
		sprite.Deallocate()
	}
	g.circleSprites = nil
}

// 圆点图像按 DPI 预先栅格化；移动时只平移，避免每帧对大量圆形重复做抗锯齿。
func (g *Game) drawDot(screen *ebiten.Image, position point, fill color.NRGBA) {
	v := screenViewport(screen)
	key := fill
	key.A = 255
	if g.dotSprites == nil {
		g.dotSprites = make(map[color.NRGBA]*ebiten.Image)
	}
	sprite := g.dotSprites[key]
	size := int(math.Ceil(22 * v.scale))
	center := float32(size) / 2
	if sprite == nil {
		sprite = ebiten.NewImage(size, size)
		vector.DrawFilledCircle(sprite, center, center+float32(2*v.scale), float32(7*v.scale), color.NRGBA{32, 45, 60, 20}, true)
		vector.DrawFilledCircle(sprite, center, center, float32(6*v.scale), key, true)
		vector.StrokeCircle(sprite, center, center, float32(6*v.scale), float32(1.5*v.scale), white, true)
		g.dotSprites[key] = sprite
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(v.x+position.x*v.scale-float64(center)), math.Round(v.y+position.y*v.scale-float64(center)))
	op.ColorScale.ScaleAlpha(float32(fill.A) / 255)
	screen.DrawImage(sprite, op)
}

package frontend

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// 白色圆形遮罩共用纹理图集，绘制时着色。工作人员、忙碌圈和选择圈均可合批。
// 内外半径使用物理像素；窗口尺寸改变时和其他纹理一起清理。
type circleKey struct{ outer, inner float64 }

func (g *Game) fillCircle(screen *ebiten.Image, x, y, radius float32, fill color.Color) {
	v := screenViewport(screen)
	g.circle(screen, x, y, circleKey{outer: float64(radius) * v.scale}, fill)
}

func (g *Game) strokeCircle(screen *ebiten.Image, x, y, radius, width float32, fill color.Color) {
	v := screenViewport(screen)
	g.circle(screen, x, y, circleKey{
		outer: float64(radius+width/2) * v.scale,
		inner: max(0, float64(radius-width/2)*v.scale),
	}, fill)
}

func circleMask(key circleKey) *image.RGBA {
	size := int(math.Ceil(key.outer*2 + 2))
	mask := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			distance := math.Hypot(float64(x)+0.5-center, float64(y)+0.5-center)
			coverage := max(0, min(1, key.outer+0.5-distance))
			if key.inner > 0 {
				coverage -= max(0, min(1, key.inner+0.5-distance))
			}
			alpha := uint8(math.Round(255 * max(0, coverage)))
			mask.SetRGBA(x, y, color.RGBA{alpha, alpha, alpha, alpha})
		}
	}
	return mask
}

func (g *Game) circle(screen *ebiten.Image, x, y float32, key circleKey, fill color.Color) {
	if key.outer <= 0 {
		return
	}
	if g.circleSprites == nil {
		g.circleSprites = make(map[circleKey]*ebiten.Image)
	}
	sprite := g.circleSprites[key]
	if sprite == nil {
		sprite = ebiten.NewImageFromImage(circleMask(key))
		g.circleSprites[key] = sprite
	}
	v := screenViewport(screen)
	center := float64(sprite.Bounds().Dx()) / 2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(v.x+float64(x)*v.scale-center), math.Round(v.y+float64(y)*v.scale-center))
	op.ColorScale.ScaleWithColor(fill)
	screen.DrawImage(sprite, op)
}

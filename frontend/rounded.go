package frontend

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// 一个小圆角纹理切成九块：四角保持原生像素，直边和中心可以伸展。
// 相同半径共用纹理，只在首次使用或 DPI 改变时创建；颜色在绘制时设置。
// DrawImage 可以合批，避免每个面板每帧执行六次矢量抗锯齿和切换渲染目标。
type roundedSprite struct {
	image *ebiten.Image
	parts [9]*ebiten.Image
}

func roundedMask(radius int) *image.RGBA {
	size := radius*2 + 1
	mask := image.NewRGBA(image.Rect(0, 0, size, size))
	if radius == 0 {
		mask.SetRGBA(0, 0, color.RGBA{255, 255, 255, 255})
		return mask
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := max(0, math.Abs(float64(x-radius))-0.5)
			dy := max(0, math.Abs(float64(y-radius))-0.5)
			alpha := uint8(math.Round(255 * max(0, min(1, float64(radius)+0.5-math.Hypot(dx, dy)))))
			mask.SetRGBA(x, y, color.RGBA{alpha, alpha, alpha, alpha})
		}
	}
	return mask
}

func (g *Game) rounded(screen *ebiten.Image, r rectangle, radius float32, fill color.Color) {
	v := screenViewport(screen)
	x, y := math.Round(v.x+float64(r.x)*v.scale), math.Round(v.y+float64(r.y)*v.scale)
	w := math.Round(v.x+float64(r.x+r.w)*v.scale) - x
	h := math.Round(v.y+float64(r.y+r.h)*v.scale) - y
	if w <= 0 || h <= 0 {
		return
	}
	pixels := int(max(0, min(math.Round(float64(radius)*v.scale), math.Floor(w/2), math.Floor(h/2))))
	if g.roundedSprites == nil {
		g.roundedSprites = make(map[int]*roundedSprite)
	}
	sprite := g.roundedSprites[pixels]
	if sprite == nil {
		sprite = &roundedSprite{image: ebiten.NewImageFromImage(roundedMask(pixels))}
		edges := [4]int{0, pixels, pixels + 1, pixels*2 + 1}
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				bounds := image.Rect(edges[col], edges[row], edges[col+1], edges[row+1])
				if !bounds.Empty() {
					sprite.parts[row*3+col] = sprite.image.SubImage(bounds).(*ebiten.Image)
				}
			}
		}
		g.roundedSprites[pixels] = sprite
	}
	edgesX := [4]float64{x, x + float64(pixels), x + w - float64(pixels), x + w}
	edgesY := [4]float64{y, y + float64(pixels), y + h - float64(pixels), y + h}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			part := sprite.parts[row*3+col]
			if part == nil || edgesX[col+1] <= edgesX[col] || edgesY[row+1] <= edgesY[row] {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale((edgesX[col+1]-edgesX[col])/float64(part.Bounds().Dx()), (edgesY[row+1]-edgesY[row])/float64(part.Bounds().Dy()))
			op.GeoM.Translate(edgesX[col], edgesY[row])
			op.ColorScale.ScaleWithColor(fill)
			screen.DrawImage(part, op)
		}
	}
}

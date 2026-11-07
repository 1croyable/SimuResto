package client

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// 优先使用系统中文字体，不把系统字体复制到仓库。没有中文字体时显示英文界面。
func loadFont(path string) (*text.GoTextFaceSource, bool, error) {
	candidates := []string{
		"C:/Windows/Fonts/msyh.ttc",
		"C:/Windows/Fonts/simhei.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/System/Library/Fonts/PingFang.ttc",
	}
	if path != "" {
		candidates = []string{path}
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if path != "" {
				return nil, false, err
			}
			continue
		}
		faces, err := text.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data))
		if err == nil && len(faces) > 0 {
			return faces[0], true, nil
		}
		if path != "" {
			return nil, false, fmt.Errorf("无法读取字体 %s: %w", path, err)
		}
	}
	face, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	return face, false, err
}

func (g *Game) label(cn, en string) string {
	if g.chinese {
		return cn
	}
	return en
}

func (g *Game) text(screen *ebiten.Image, value string, x, y, size float64, ink color.Color) {
	v := screenViewport(screen)
	op := &text.DrawOptions{}
	op.GeoM.Translate(math.Round(v.x+x*v.scale), math.Round(v.y+y*v.scale))
	op.ColorScale.ScaleWithColor(ink)
	// 按实际像素字号生成字形；不对已栅格化的文字做二次缩放。
	text.Draw(screen, value, &text.GoTextFace{Source: g.font, Size: size * v.scale}, op)
}

func (g *Game) fit(value string, size, width float64) string {
	face := &text.GoTextFace{Source: g.font, Size: size}
	if w, _ := text.Measure(value, face, 0); w <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if w, _ := text.Measure(string(runes)+"…", face, 0); w <= width {
			return string(runes) + "…"
		}
	}
	return ""
}

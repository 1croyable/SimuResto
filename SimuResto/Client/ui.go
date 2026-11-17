package client

import (
	"image/color"

	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

type ScreenMode int

const (
	Home ScreenMode = iota
	Running
	Paused
)

const (
	ButtonX  = 470
	ButtonW  = 260
	ButtonH  = 48
	Button1Y = 325
	Button2Y = 395
	Button3Y = 465
)

func MenuButtonHit(x, y, buttonY int) bool {
	return x >= buttonX && x <= buttonX+buttonW && y >= buttonY && y <= buttonY+buttonH
}

func DrawButton(screen *ebiten.Image, label string, y int) {
	vector.FillRect(screen, buttonX, float32(y), buttonW, buttonH, color.RGBA{R: 205, G: 222, B: 240, A: 255}, false)

	textX := buttonX + (buttonW-len(label)*6)/2
	DrawMenuText(screen, label, textX, y+18)
}

func DrawMenu(screen *ebiten.Image, mode ScreenMode) {
	if mode == Paused {
		// darken
		vector.FillRect(screen, 0, 0, environment.Width, environment.Height, color.RGBA{R: 0, G: 0, B: 0, A: 160}, false)
	}

	// Menu panel
	vector.FillRect(screen, 390, 180, 420, 390, color.RGBA{R: 245, G: 245, B: 245, A: 255}, false)

	if mode == Home {
		DrawMenuText(screen, "SIMURESTO", 565, 245)
		DrawButton(screen, "Start Simulation", button1Y)
		DrawButton(screen, "Quit", button2Y)
	} else {
		DrawMenuText(screen, "PAUSED", 580, 245)
		DrawButton(screen, "Resume Simulation", button1Y)
		DrawButton(screen, "Restart Simulation", button2Y)
		DrawButton(screen, "Main Menu", button3Y)
	}
}

var menuFont = text.NewGoXFace(basicfont.Face7x13)

func DrawMenuText(screen *ebiten.Image, label string, x, y int) {
	draw := func(dx, dy float64, clr color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x)+dx, float64(y)+dy)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, label, menuFont, op)
	}

	draw(-1, 0, color.White)
	draw(1, 0, color.White)
	draw(0, -1, color.White)
	draw(0, 1, color.White)

	draw(0, 0, color.Black)
}

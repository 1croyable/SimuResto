package main

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
	buttonX  = 470
	buttonW  = 260
	buttonH  = 48
	button1Y = 325
	button2Y = 395
	button3Y = 465
)

func menuButtonHit(x, y, buttonY int) bool {
	return x >= buttonX && x <= buttonX+buttonW && y >= buttonY && y <= buttonY+buttonH
}

func drawButton(screen *ebiten.Image, label string, y int) {
	vector.FillRect(screen, buttonX, float32(y), buttonW, buttonH, color.RGBA{R: 205, G: 222, B: 240, A: 255}, false)

	textX := buttonX + (buttonW-len(label)*6)/2
	drawMenuText(screen, label, textX, y+18)
}

func drawMenu(screen *ebiten.Image, mode ScreenMode) {
	if mode == Paused {
		// darken
		vector.FillRect(screen, 0, 0, environment.Width, environment.Height, color.RGBA{R: 0, G: 0, B: 0, A: 160}, false)
	}

	// Menu panel
	vector.FillRect(screen, 390, 180, 420, 390, color.RGBA{R: 245, G: 245, B: 245, A: 255}, false)

	if mode == Home {
		drawMenuText(screen, "SIMURESTO", 565, 245)
		drawButton(screen, "Start Simulation", button1Y)
		drawButton(screen, "Quit", button2Y)
	} else {
		drawMenuText(screen, "PAUSED", 580, 245)
		drawButton(screen, "Resume Simulation", button1Y)
		drawButton(screen, "Restart Simulation", button2Y)
		drawButton(screen, "Main Menu", button3Y)
	}
}

var menuFont = text.NewGoXFace(basicfont.Face7x13)

func drawMenuText(screen *ebiten.Image, label string, x, y int) {
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

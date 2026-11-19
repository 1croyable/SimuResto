package client

import (
	"fmt"
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
	Configuring
)

const (
	ButtonX  = 470
	ButtonW  = 260
	ButtonH  = 48
	Button1Y = 325
	Button2Y = 395
	Button3Y = 465

	PauseMinusX   = 455
	PausePlusX    = 695
	PauseControlY = 313
	PauseControlW = 50
	PauseControlH = 44
)

var pauseSpeedChoices = [...]float64{0.5, 1, 2, 5, 10, 20, 50, 100}

// AdjustPauseSpeed changes the selected speed one preset at a time.
// direction must be -1 or +1.
func AdjustPauseSpeed(current float64, direction int) float64 {
	if direction == 0 {
		return current
	}
	if direction < 0 {
		for i := len(pauseSpeedChoices) - 1; i >= 0; i-- {
			if pauseSpeedChoices[i] < current-0.00001 {
				return pauseSpeedChoices[i]
			}
		}
		return pauseSpeedChoices[0]
	}
	for _, speed := range pauseSpeedChoices {
		if speed > current+0.00001 {
			return speed
		}
	}
	return pauseSpeedChoices[len(pauseSpeedChoices)-1]
}

func rectHit(x, y, left, top, width, height int) bool {
	return x >= left && x < left+width && y >= top && y < top+height
}

func MenuButtonHit(x, y, buttonY int) bool {
	return rectHit(x, y, ButtonX, buttonY, ButtonW, ButtonH)
}

func PauseDecreaseHit(x, y int) bool {
	return rectHit(x, y, PauseMinusX, PauseControlY, PauseControlW, PauseControlH)
}

func PauseIncreaseHit(x, y int) bool {
	return rectHit(x, y, PausePlusX, PauseControlY, PauseControlW, PauseControlH)
}

func DrawButton(screen *ebiten.Image, label string, y int) {
	vector.FillRect(screen, ButtonX, float32(y), ButtonW, ButtonH,
		color.RGBA{R: 205, G: 222, B: 240, A: 255}, false)
	textX := ButtonX + (ButtonW-len(label)*7)/2
	DrawMenuText(screen, label, textX, y+18)
}

func DrawMenu(screen *ebiten.Image, mode ScreenMode) {
	if mode == Paused {
		DrawPauseMenu(screen, 10)
		return
	}
	vector.FillRect(screen, 390, 180, 420, 390,
		color.RGBA{R: 245, G: 245, B: 245, A: 255}, false)
	DrawMenuText(screen, "SIMURESTO", 565, 245)
	DrawButton(screen, "Start Simulation", Button1Y)
	DrawButton(screen, "Quit", Button2Y)
}

func DrawPauseMenu(screen *ebiten.Image, speed float64) {
	vector.FillRect(screen, 0, 0, environment.Width, environment.Height,
		color.RGBA{R: 0, G: 0, B: 0, A: 170}, false)
	vector.FillRect(screen, 390, 150, 420, 470,
		color.RGBA{R: 245, G: 245, B: 245, A: 255}, false)

	DrawMenuText(screen, "SIMULATION PAUSED", 525, 197)
	DrawMenuText(screen, "Simulation speed (simulated / real time)", 459, 274)

	buttonFill := color.RGBA{R: 205, G: 222, B: 240, A: 255}
	vector.FillRect(screen, PauseMinusX, PauseControlY, PauseControlW, PauseControlH, buttonFill, false)
	vector.FillRect(screen, PausePlusX, PauseControlY, PauseControlW, PauseControlH, buttonFill, false)
	DrawMenuText(screen, "-", PauseMinusX+22, PauseControlY+15)
	DrawMenuText(screen, "+", PausePlusX+22, PauseControlY+15)
	DrawMenuText(screen, fmt.Sprintf("%g x", speed), 579, PauseControlY+15)

	DrawButton(screen, "Resume Simulation", Button1Y+65)
	DrawButton(screen, "Restart Simulation", Button2Y+65)
	DrawButton(screen, "Main Menu", Button3Y+65)
	DrawMenuText(screen, "Changes apply when you resume", 497, 597)
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

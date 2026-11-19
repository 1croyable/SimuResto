package client

import (
	"fmt"
	"image/color"
	"strings"

	environment "simuresto/SimuResto/Environment"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type SettingsPage struct {
	settings      *environment.Settings
	backup        environment.Settings
	fields        []environment.SettingField
	active        int
	firstVisible  int
	buffer        string
	replaceOnType bool
	errorMessage  string
}

const (
	configX     = 170
	configY     = 28
	configW     = 860
	configH     = 644
	configRowsY = 137
	configRowH  = 29
	configRows  = 14

	configSaveX   = 405
	configCancelX = 640
	configButtonY = 602
	configButtonW = 170
	configButtonH = 45
)

func NewSettingsPage(settings *environment.Settings) *SettingsPage {
	return &SettingsPage{
		settings: settings,
		backup:   *settings,
		fields:   settings.Fields(),
		active:   -1,
	}
}

func pointInRect(px, py, x, y, width, height int) bool {
	return px >= x && py >= y && px < x+width && py < y+height
}

// Commit the edited string through Environment.Settings.SetField.
func (p *SettingsPage) commit() bool {
	if p.active < 0 {
		return true
	}
	field := p.fields[p.active]
	if err := p.settings.SetField(field.Key, p.buffer); err != nil {
		p.errorMessage = err.Error()
		return false
	}
	p.fields = p.settings.Fields()
	p.errorMessage = ""
	return true
}

func (p *SettingsPage) focus(index int) {
	if index < 0 || index >= len(p.fields) || index == p.active {
		return
	}
	if !p.commit() {
		return
	}
	p.active = index
	p.buffer = p.fields[index].Value
	p.replaceOnType = true
	if index < p.firstVisible {
		p.firstVisible = index
	}
	if index >= p.firstVisible+configRows {
		p.firstVisible = index - configRows + 1
	}
}

// Update returns (saved, canceled). Only Save begins the simulation.
func (p *SettingsPage) Update() (bool, bool) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*p.settings = p.backup
		return false, true
	}

	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		p.firstVisible -= 3
	} else if wheelY < 0 {
		p.firstVisible += 3
	}
	p.firstVisible = max(0, min(p.firstVisible, max(0, len(p.fields)-configRows)))

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if pointInRect(x, y, configSaveX, configButtonY, configButtonW, configButtonH) {
			if !p.commit() {
				return false, false
			}
			if err := p.settings.Validate(); err != nil {
				p.errorMessage = err.Error()
				return false, false
			}
			return true, false
		}

		if pointInRect(x, y, configCancelX, configButtonY, configButtonW, configButtonH) {
			*p.settings = p.backup
			return false, true
		}

		if pointInRect(x, y, configX+24, configRowsY, configW-48, configRows*configRowH) {
			index := p.firstVisible + (y-configRowsY)/configRowH
			p.focus(index)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		p.focus((p.active + 1) % len(p.fields))
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		index := p.active - 1
		if index < 0 {
			index = len(p.fields) - 1
		}
		p.focus(index)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && p.active >= 0 {
		p.focus((p.active + 1) % len(p.fields))
	}

	if p.active < 0 {
		return false, false
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if p.replaceOnType {
			p.buffer = ""
		} else {
			chars := []rune(p.buffer)
			if len(chars) > 0 {
				p.buffer = string(chars[:len(chars)-1])
			}
		}
		p.replaceOnType = false
		p.errorMessage = ""
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		p.buffer = ""
		p.replaceOnType = false
		p.errorMessage = ""
	}

	for _, char := range ebiten.AppendInputChars(nil) {
		// All fields are numeric, durations or times. No free-form values.
		if !strings.ContainsRune("0123456789.-:+hmsunµ", char) {
			continue
		}
		if p.replaceOnType {
			p.buffer = ""
			p.replaceOnType = false
		}
		p.buffer += string(char)
		p.errorMessage = ""
	}

	return false, false
}

func (p *SettingsPage) Draw(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, environment.Width, environment.Height,
		color.RGBA{R: 21, G: 30, B: 49, A: 195}, false)

	vector.FillRect(screen, configX, configY, configW, configH,
		color.RGBA{R: 240, G: 242, B: 248, A: 255}, false)
	vector.FillRect(screen, configX, configY, configW, 70,
		color.RGBA{R: 35, G: 57, B: 88, A: 255}, false)

	DrawMenuText(screen, "SIMULATION SETTINGS", configX+330, configY+22)
	DrawMenuText(screen, "Click a value to edit  |  Tab / Enter: next field  |  Mouse wheel: scroll", configX+170, configY+51)
	DrawMenuText(screen, "PARAMETER", configX+50, configRowsY-19)
	DrawMenuText(screen, "VALUE", configX+530, configRowsY-19)

	for row := 0; row < configRows; row++ {
		index := p.firstVisible + row
		if index >= len(p.fields) {
			break
		}
		field := p.fields[index]
		y := configRowsY + row*configRowH
		fill := color.RGBA{R: 222, G: 231, B: 243, A: 255}
		if index%2 == 1 {
			fill = color.RGBA{R: 232, G: 238, B: 246, A: 255}
		}
		if index == p.active {
			fill = color.RGBA{R: 169, G: 211, B: 254, A: 255}
		}
		vector.FillRect(screen, configX+24, float32(y), configW-48, configRowH-2, fill, false)
		vector.FillRect(screen, configX+497, float32(y+2), 282, configRowH-6,
			color.White, false)

		DrawMenuText(screen, field.Label, configX+43, y+7)
		value := field.Value
		if index == p.active {
			value = p.buffer
		}
		if len(value) > 38 {
			value = value[:38]
		}
		DrawMenuText(screen, value, configX+512, y+7)
	}

	DrawMenuText(screen, fmt.Sprintf("Fields %d - %d of %d",
		p.firstVisible+1, min(p.firstVisible+configRows, len(p.fields)), len(p.fields)),
		configX+42, 551)
	if p.active >= 0 {
		DrawMenuText(screen, "Format: "+p.fields[p.active].Hint, configX+290, 551)
	}
	if p.errorMessage != "" {
		message := p.errorMessage
		if len(message) > 103 {
			message = message[:103]
		}
		DrawMenuText(screen, message, configX+42, 576)
	}

	vector.FillRect(screen, configSaveX, configButtonY, configButtonW, configButtonH,
		color.RGBA{R: 110, G: 200, B: 128, A: 255}, false)
	vector.FillRect(screen, configCancelX, configButtonY, configButtonW, configButtonH,
		color.RGBA{R: 229, G: 144, B: 131, A: 255}, false)
	DrawMenuText(screen, "SAVE & START", configSaveX+35, configButtonY+14)
	DrawMenuText(screen, "CANCEL", configCancelX+62, configButtonY+14)
}

package client

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

type settingKey int

const (
	studentCount settingKey = iota
	cashierCount
	tickSeconds
	maxMinutes
)

type settingSpec struct {
	cn, en, unit           string
	minimum, maximum, step float64
	integer                bool
}

// 开始前只保留四个数值设置，其他食堂规则由 Environment 中的常量决定。
var settingSpecs = []settingSpec{
	{"学生人数", "Students", "", 1, 2000, 10, true},
	{"收银窗口数量", "Open cashiers", "", 1, 2, 1, true},
	{"逻辑步长", "Logical step", "s", 0.05, 2, 0.05, false},
	{"最长模拟时间", "Maximum duration", "min", 1, 120, 1, true},
}

var settingTabs = [][2]string{
	{"01  模拟设置", "01  Simulation"},
	{"02  主菜备餐", "02  Prepare mains"},
}

var startButton = rectangle{1100, 782, 300, 54}
var defaultsButton = rectangle{1100, 716, 300, 44}

func settingTabBox(index int) rectangle {
	return rectangle{48 + float32(index)*244, 177, 232, 43}
}

func settingBox(index int) rectangle {
	return rectangle{48 + float32(index%2)*510, 280 + float32(index/2)*112, 344, 46}
}

func settingText(value float64) string {
	return strconv.FormatFloat(math.Round(value*1e6)/1e6, 'f', -1, 64)
}

func (g *Game) populateSettings() {
	g.settingValues = []string{
		strconv.Itoa(len(g.options.Students)),
		strconv.Itoa(g.options.Config.Cashiers),
		settingText(g.options.Config.Tick.Seconds()),
		settingText(g.options.Config.MaxDuration.Minutes()),
	}
	g.focusedSetting, g.settingsError = -1, ""
	g.populatePreparation()
}

func (g *Game) openSettings() {
	g.Close()
	g.configuring = true
	g.populateSettings()
}

func (g *Game) restoreDefaults() {
	g.options.Config = environment.DefaultConfig()
	g.speed = environment.DefaultPlaybackSpeed
	generator := agents.StudentGenerator{Seed: environment.RandomSeed, ArrivalInterval: environment.ArrivalInterval}
	g.options.Students, _ = generator.Generate(environment.DefaultStudentCount)
	g.populateSettings()
}

func (g *Game) startConfigured() error {
	values := make([]float64, len(settingSpecs))
	for i, spec := range settingSpecs {
		value, err := strconv.ParseFloat(g.settingValues[i], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < spec.minimum || value > spec.maximum || (spec.integer && value != math.Trunc(value)) {
			g.settingsTab, g.focusedSetting = 0, i
			kind := g.label("数值", "number")
			if spec.integer {
				kind = g.label("整数", "integer")
			}
			return fmt.Errorf(g.label("%s：请输入 %g–%g 之间的%s", "%s: enter %g–%g (%s)"),
				g.label(spec.cn, spec.en), spec.minimum, spec.maximum, kind)
		}
		values[i] = value
	}
	mains, err := g.preparedMainDishes()
	if err != nil {
		g.settingsTab = preparationTab
		return err
	}
	config := environment.DefaultConfig()
	config.Tick = time.Duration(values[tickSeconds] * float64(time.Second))
	config.MaxDuration = time.Duration(values[maxMinutes] * float64(time.Minute))
	config.Cashiers = int(values[cashierCount])
	// 前菜和甜点使用固定菜单，主菜采用备餐页逐行指定的组合和数量。
	config.Menu = nil
	for _, dish := range environment.DefaultMenu() {
		if dish.Course != agents.Main {
			config.Menu = append(config.Menu, dish)
		}
	}
	config.Menu = append(config.Menu, mains...)
	generator := agents.StudentGenerator{Seed: environment.RandomSeed, ArrivalInterval: environment.ArrivalInterval}
	students, err := generator.Generate(int(values[studentCount]))
	if err != nil {
		return err
	}
	if err := config.Validate(students); err != nil {
		return err
	}
	g.options.Config, g.options.Students = config, students
	return g.reset()
}

func adjustNumber(value *string, minimum, maximum, step float64) {
	number, err := strconv.ParseFloat(*value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		number = minimum
	}
	*value = settingText(max(minimum, min(maximum, number+step)))
}

// 设置值和备餐份数共用数字输入，避免维护两套键盘处理逻辑。
func editNumber(value *string, replace *bool, integer bool) {
	if (ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
		*replace = true
	}
	for _, char := range ebiten.AppendInputChars(nil) {
		if (char >= '0' && char <= '9') || (char == '.' && !integer) {
			if *replace {
				*value, *replace = "", false
			}
			if len(*value) < 8 {
				*value += string(char)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if *replace {
			*value = ""
		} else if len(*value) > 0 {
			*value = (*value)[:len(*value)-1]
		}
		*replace = false
	}
}

func (g *Game) changeSetting(index int, direction float64) {
	spec := settingSpecs[index]
	adjustNumber(&g.settingValues[index], spec.minimum, spec.maximum, direction*spec.step)
	g.settingsError = ""
}

func (g *Game) updateSettings() {
	x, y := g.cursor()
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if g.ingredientPicker >= 0 {
		if clicked {
			g.chooseIngredient(x, y)
		}
		return
	}
	if clicked {
		g.focusedSetting = -1
		for tab := range settingTabs {
			if settingTabBox(tab).contains(x, y) {
				g.settingsTab, g.preparationFocus = tab, -1
				return
			}
		}
		if defaultsButton.contains(x, y) {
			g.restoreDefaults()
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || (clicked && startButton.contains(x, y)) {
		if err := g.startConfigured(); err != nil {
			g.settingsError = err.Error()
		}
		return
	}
	if g.settingsTab == preparationTab {
		g.updatePreparation(x, y, clicked)
		return
	}
	if clicked {
		for i := range settingSpecs {
			box := settingBox(i)
			if box.contains(x, y) {
				g.focusedSetting, g.replaceSetting = i, true
			}
			if (rectangle{box.x + 354, box.y, 46, 46}).contains(x, y) {
				g.changeSetting(i, -1)
			}
			if (rectangle{box.x + 410, box.y, 46, 46}).contains(x, y) {
				g.changeSetting(i, 1)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.focusedSetting = (g.focusedSetting + 1) % len(settingSpecs)
		g.replaceSetting = true
	}
	index := g.focusedSetting
	if index < 0 {
		return
	}
	before := g.settingValues[index]
	editNumber(&g.settingValues[index], &g.replaceSetting, settingSpecs[index].integer)
	if before != g.settingValues[index] {
		g.settingsError = ""
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.changeSetting(index, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.changeSetting(index, -1)
	}
}

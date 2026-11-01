package frontend

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"simuresto/simulation"
)

type settingKey int

const (
	studentCount settingKey = iota
	arrivalSeconds
	randomSeed
	outdoorCapacity
	indoorCapacity
	selfCapacity
	mainCapacity
	cashierCount
	serverCount
	selfSeconds
	mainSeconds
	paymentSeconds
	starterStock
	vegetarianStock
	meatStock
	dessertStock
	tickSeconds
	maxMinutes
	playbackSpeed
	indoorLoss
	outdoorLoss
	starterLimit
	dessertLimit
	totalSideLimit
	mainWeight
	starterWeight
	dessertWeight
	studentTemperature
)

type settingSpec struct {
	cn, en, unit           string
	minimum, maximum, step float64
	integer                bool
}

// 每页最多八项；范围同时用于按钮步进和开始前校验。
var settingSpecs = []settingSpec{
	{"学生人数", "Students", "", 1, 2000, 10, true},
	{"到达间隔", "Arrival interval", "s", 0, 60, 0.25, false},
	{"随机种子", "Random seed", "", 0, 2147483647, 1, true},
	{"室外队列容量", "Outdoor capacity", "", 1, 200, 5, true},
	{"室内队列容量", "Indoor capacity", "", 1, 60, 5, true},
	{"自助区容量", "Self-service capacity", "", 1, 18, 1, true},
	{"主菜区容量", "Main-service capacity", "", 1, 12, 1, true},
	{"收银窗口数量", "Payment windows", "", 1, 2, 1, true},
	{"打饭工作人员", "Main-course staff", "", 1, 6, 1, true},
	{"自助取餐耗时", "Self-service time", "s", 0.25, 120, 0.25, false},
	{"主菜服务耗时", "Main service time", "s", 0.25, 120, 0.25, false},
	{"付款耗时", "Payment time", "s", 0.25, 120, 0.25, false},
	{"前菜总库存", "Starter portions", "", 0, 10000, 10, true},
	{"素食主菜总库存", "Vegetarian main portions", "", 0, 10000, 10, true},
	{"荤食主菜总库存", "Meat main portions", "", 0, 10000, 10, true},
	{"甜点总库存", "Dessert portions", "", 0, 10000, 10, true},
	{"逻辑时钟步长", "Logical clock step", "s", 0.05, 2, 0.05, false},
	{"最长模拟时间", "Maximum duration", "min", 1, 120, 1, true},
	{"播放速度", "Playback speed", "x", 0.25, 32, 0.25, false},
	{"室内每秒扣分", "Indoor loss / second", "", 0, 5, 0.01, false},
	{"室外每秒扣分", "Outdoor loss / second", "", 0, 10, 0.01, false},
	{"每人前菜上限", "Starter allowance", "", 0, 5, 1, true},
	{"每人甜点上限", "Dessert allowance", "", 0, 5, 1, true},
	{"每人配菜总上限", "Total side allowance", "", 0, 10, 1, true},
	{"主菜满意度权重", "Main score weight", "", 0, 10, 0.25, false},
	{"前菜满意度权重", "Starter score weight", "", 0, 10, 0.25, false},
	{"甜点满意度权重", "Dessert score weight", "", 0, 10, 0.25, false},
	{"学生选菜温度", "Choice temperature", "", 0, 10, 0.1, false},
}

var settingTabs = [][2]string{
	{"01  人流与队列", "01  Arrivals & queues"},
	{"02  服务与库存", "02  Service & stock"},
	{"03  行为与时钟", "03  Behavior & clock"},
	{"04  菜品满意度", "04  Meal scores"},
}

func settingTabBox(index int) rectangle {
	return rectangle{48 + float32(index)*246, 177, 238, 43}
}

var startButton = rectangle{1100, 782, 300, 54}
var defaultsButton = rectangle{1100, 716, 300, 44}

func settingBox(index int) rectangle {
	index %= 8
	return rectangle{48 + float32(index%2)*510, 280 + float32(index/2)*112, 344, 46}
}

func settingText(value float64) string {
	return strconv.FormatFloat(math.Round(value*1e6)/1e6, 'f', -1, 64)
}

func (g *Game) populateSettings() {
	c := g.options.Config
	temperature := simulation.DefaultTemperature
	if len(g.options.Students) > 0 {
		temperature = g.options.Students[0].Temperature
	}
	values := []float64{
		float64(len(g.options.Students)), g.options.ArrivalInterval.Seconds(), float64(c.Seed),
		float64(c.Capacities.Outside), float64(c.Capacities.Inside), float64(c.Capacities.SelfService), float64(c.Capacities.MainService), float64(c.Capacities.Payment),
		float64(c.Servers), c.SelfServiceTime.Seconds(), c.MainServiceTime.Seconds(), c.PaymentTime.Seconds(),
		0, 0, 0, 0,
		c.Tick.Seconds(), c.MaxDuration.Minutes(), g.speed, float64(c.IndoorLossPerSecond), float64(c.OutdoorLossPerSecond),
		float64(c.MaxStartersPerStudent), float64(c.MaxDessertsPerStudent), float64(c.MaxSelfServiceItems),
		float64(c.SatisfactionWeights.Main), float64(c.SatisfactionWeights.Starter), float64(c.SatisfactionWeights.Dessert),
		float64(temperature),
	}
	for _, dish := range c.Menu {
		switch dish.Course {
		case simulation.Starter:
			values[starterStock] += float64(dish.Portions)
		case simulation.Dessert:
			values[dessertStock] += float64(dish.Portions)
		case simulation.Main:
			if dish.Vegetarian {
				values[vegetarianStock] += float64(dish.Portions)
			} else {
				values[meatStock] += float64(dish.Portions)
			}
		}
	}
	g.settingValues = make([]string, len(values))
	for i, value := range values {
		g.settingValues[i] = settingText(value)
	}
	g.focusedSetting, g.settingsError = -1, ""
}

func (g *Game) openSettings() {
	g.Close()
	g.configuring = true
	g.populateSettings()
}

func (g *Game) restoreDefaults() {
	g.options.Config = simulation.DefaultConfig()
	g.options.Config.Tick = 250 * time.Millisecond
	g.options.Config.Capacities.Payment, g.options.Config.Cashiers = 2, 2
	g.options.ArrivalInterval, g.speed = time.Second, 4
	generator := simulation.StudentGenerator{
		Seed:            g.options.Config.Seed,
		ArrivalInterval: time.Second,
	}
	g.options.Students, _ = generator.Generate(100)
	g.populateSettings()
}

// 库存参数是每组的总量，按菜单顺序平均分配，保持配置与实际库存一致。
func distributePortions(menu []simulation.Dish, course simulation.Course, vegetarian bool, total int) {
	var indices []int
	for i, dish := range menu {
		if dish.Course == course && (course != simulation.Main || dish.Vegetarian == vegetarian) {
			indices = append(indices, i)
		}
	}
	for i, index := range indices {
		menu[index].Portions = total / len(indices)
		if i < total%len(indices) {
			menu[index].Portions++
		}
	}
}

func (g *Game) startConfigured() error {
	values := make([]float64, len(settingSpecs))
	for i, spec := range settingSpecs {
		value, err := strconv.ParseFloat(g.settingValues[i], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < spec.minimum || value > spec.maximum || (spec.integer && value != math.Trunc(value)) {
			g.settingsTab, g.focusedSetting = i/8, i
			return fmt.Errorf(g.label("%s：请输入 %g–%g 之间的%s", "%s: enter %g–%g (%s)"), g.label(spec.cn, spec.en), spec.minimum, spec.maximum, g.label(map[bool]string{true: "整数", false: "数值"}[spec.integer], map[bool]string{true: "integer", false: "number"}[spec.integer]))
		}
		values[i] = value
	}
	if values[outdoorLoss] <= values[indoorLoss] {
		g.settingsTab = 2
		return fmt.Errorf("%s", g.label("室外每秒扣分必须大于室内每秒扣分。", "Outdoor loss must exceed indoor loss."))
	}
	if values[mainWeight]+values[starterWeight]+values[dessertWeight] == 0 {
		g.settingsTab, g.focusedSetting = int(mainWeight)/8, int(mainWeight)
		return fmt.Errorf("%s", g.label("主菜、前菜和甜点的权重不能全为 0。", "At least one meal score weight must be positive."))
	}
	c := g.options.Config
	c.Menu = append([]simulation.Dish(nil), c.Menu...)
	c.Seed = int64(values[randomSeed])
	c.Capacities = simulation.Capacities{Outside: int(values[outdoorCapacity]), Inside: int(values[indoorCapacity]), SelfService: int(values[selfCapacity]), MainService: int(values[mainCapacity]), Payment: int(values[cashierCount])}
	c.Cashiers, c.Servers = int(values[cashierCount]), int(values[serverCount])
	c.SelfServiceTime = time.Duration(values[selfSeconds] * float64(time.Second))
	c.MainServiceTime = time.Duration(values[mainSeconds] * float64(time.Second))
	c.PaymentTime = time.Duration(values[paymentSeconds] * float64(time.Second))
	c.Tick = time.Duration(values[tickSeconds] * float64(time.Second))
	c.MaxDuration = time.Duration(values[maxMinutes] * float64(time.Minute))
	c.IndoorLossPerSecond, c.OutdoorLossPerSecond = float32(values[indoorLoss]), float32(values[outdoorLoss])
	c.MaxStartersPerStudent, c.MaxDessertsPerStudent, c.MaxSelfServiceItems = int(values[starterLimit]), int(values[dessertLimit]), int(values[totalSideLimit])
	c.SatisfactionWeights = simulation.CourseWeights{
		Main: float32(values[mainWeight]), Starter: float32(values[starterWeight]), Dessert: float32(values[dessertWeight]),
	}
	distributePortions(c.Menu, simulation.Starter, true, int(values[starterStock]))
	distributePortions(c.Menu, simulation.Main, true, int(values[vegetarianStock]))
	distributePortions(c.Menu, simulation.Main, false, int(values[meatStock]))
	distributePortions(c.Menu, simulation.Dessert, true, int(values[dessertStock]))
	interval := time.Duration(values[arrivalSeconds] * float64(time.Second))
	generator := simulation.StudentGenerator{
		Seed:            c.Seed,
		ArrivalInterval: interval,
	}
	students, err := generator.Generate(int(values[studentCount]))
	if err != nil {
		return err
	}
	for i := range students {
		students[i].Temperature = float32(values[studentTemperature])
	}
	g.options.Config, g.options.Students, g.options.ArrivalInterval = c, students, interval
	g.speed = values[playbackSpeed]
	if err := g.reset(); err != nil {
		return err
	}
	g.paused = g.options.Paused
	return nil
}

func (g *Game) changeSetting(index int, direction float64) {
	spec := settingSpecs[index]
	value, err := strconv.ParseFloat(g.settingValues[index], 64)
	if err != nil {
		value = spec.minimum
	}
	value = max(spec.minimum, min(spec.maximum, value+direction*spec.step))
	g.settingValues[index] = settingText(value)
	g.settingsError = ""
}

func (g *Game) updateSettings() {
	x, y := g.cursor()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.focusedSetting = -1
		for tab := range settingTabs {
			if settingTabBox(tab).contains(x, y) {
				g.settingsTab = tab
			}
		}
		for i := g.settingsTab * 8; i < min(g.settingsTab*8+8, len(settingSpecs)); i++ {
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
		if defaultsButton.contains(x, y) {
			g.restoreDefaults()
		}
		if startButton.contains(x, y) {
			if err := g.startConfigured(); err != nil {
				g.settingsError = err.Error()
			}
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		start, end := g.settingsTab*8, min(g.settingsTab*8+8, len(settingSpecs))
		if g.focusedSetting < start || g.focusedSetting >= end {
			g.focusedSetting = start
		} else {
			g.focusedSetting = start + (g.focusedSetting-start+1)%(end-start)
		}
		g.replaceSetting = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if err := g.startConfigured(); err != nil {
			g.settingsError = err.Error()
		}
		return
	}
	index := g.focusedSetting
	if index < 0 {
		return
	}
	if (ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.replaceSetting = true
	}
	for _, char := range ebiten.AppendInputChars(nil) {
		if (char >= '0' && char <= '9') || char == '.' || char == '-' {
			if g.replaceSetting {
				g.settingValues[index] = ""
				g.replaceSetting = false
			}
			if len(g.settingValues[index]) < 14 {
				g.settingValues[index] += string(char)
			}
			g.settingsError = ""
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		value := g.settingValues[index]
		if g.replaceSetting {
			value = ""
		} else if len(value) > 0 {
			value = value[:len(value)-1]
		}
		g.settingValues[index], g.replaceSetting = value, false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.changeSetting(index, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.changeSetting(index, -1)
	}
}

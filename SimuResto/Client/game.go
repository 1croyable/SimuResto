package client

import (
	"context"
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	agents "simuresto/SimuResto/Agents"
	engine "simuresto/SimuResto/Engine"
	environment "simuresto/SimuResto/Environment"
)

type Options struct {
	Config   environment.Config
	Students []agents.Student
}

type dot struct {
	id         int
	position   point
	path       []point
	area       agents.Area
	status     agents.Status
	preference agents.Preference
	fade       float64
}

type Game struct {
	options            Options
	simulationGame     *engine.Game
	snapshot           engine.Snapshot
	font               *text.GoTextFaceSource
	chinese            bool
	paused             bool
	speed              float64
	accumulator        float64
	dots               map[int]*dot
	selected           int
	notice             string
	viewport           viewport
	configuring        bool
	settingValues      []string
	settingsTab        int
	focusedSetting     int
	replaceSetting     bool
	settingsError      string
	preparation        []preparationRow
	preparationPage    int
	preparationFocus   int
	ingredientPicker   int // -1 表示关闭；其余为行号 × 3 + 食材列号。
	preparationReplace bool
	frame              frameLimiter
	floorLayer         *ebiten.Image
	dotSprites         map[color.NRGBA]*ebiten.Image
	renderSize         image.Point
	roundedSprites     map[int]*roundedSprite
	circleSprites      map[circleKey]*ebiten.Image
}

func New(options Options) (*Game, error) {
	if err := options.Config.Validate(options.Students); err != nil {
		return nil, err
	}
	font, chinese, err := loadFont()
	if err != nil {
		return nil, err
	}
	game := &Game{
		options: options, font: font, chinese: chinese,
		speed: environment.DefaultPlaybackSpeed, configuring: true,
	}
	game.populateSettings()
	return game, nil
}

func (g *Game) reset() error {
	g.clearRenderCache()
	if g.simulationGame != nil {
		g.simulationGame.Close()
	}
	simulationGame, err := engine.NewGame(g.options.Config, g.options.Students)
	if err != nil {
		return err
	}
	g.simulationGame = simulationGame
	g.snapshot = simulationGame.Snapshot()
	g.dots = make(map[int]*dot)
	g.accumulator, g.selected, g.notice = 0, 0, ""
	g.paused = false
	g.configuring = false
	return nil
}

func (g *Game) Close() {
	if g.simulationGame != nil {
		g.simulationGame.Close()
	}
}

func (g *Game) step() {
	_, err := g.simulationGame.Step(context.Background())
	if err != nil && !errors.Is(err, engine.ErrTimeLimit) {
		g.notice = err.Error()
	}
	if errors.Is(err, engine.ErrTimeLimit) {
		g.notice = g.label("达到模拟时限", "Time limit reached")
	}
	g.snapshot = g.simulationGame.Snapshot()
	g.syncDots()
}

// 切换收银只更新当前模拟，不重新生成学生或清空库存。
func (g *Game) toggleCashiers() error {
	count := 3 - g.options.Config.Cashiers
	if err := g.simulationGame.SetCashiers(count); err != nil {
		return err
	}
	g.options.Config.Cashiers = count
	g.snapshot = g.simulationGame.Snapshot()
	g.syncDots()
	g.notice = ""
	return nil
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.configuring && g.ingredientPicker >= 0 {
			g.ingredientPicker = -1
			return nil
		}
		return ebiten.Termination
	}
	if g.configuring {
		g.updateSettings()
		return nil
	}
	x, y := g.cursor()
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || (clicked && pauseButton.contains(x, y)) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || (clicked && restartButton.contains(x, y)) {
		g.openSettings()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if err := g.reset(); err != nil {
			return err
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) || (clicked && cashierButton.contains(x, y)) {
		if err := g.toggleCashiers(); err != nil {
			g.notice = err.Error()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || (clicked && speedButton.contains(x, y)) {
		g.speed = min(32, g.speed*2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.speed = max(0.25, g.speed/2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) || (clicked && stepButton.contains(x, y)) {
		g.paused = true
		if !g.snapshot.Done {
			g.step()
			g.animate(g.options.Config.Tick.Seconds())
		}
	}
	if clicked && y > 100 && x < 1064 {
		g.selected = 0
		best := 15.0
		for id, person := range g.dots {
			distance := math.Hypot(person.position.x-float64(x), person.position.y-float64(y))
			if distance < best {
				best, g.selected = distance, id
			}
		}
	}
	if !g.paused {
		delta := g.speed / FPSLimit
		g.accumulator += delta
		// 帧率与逻辑时钟分开；每帧最多推进 256 步，避免极小 tick 阻塞窗口。
		for steps := 0; steps < 256 && !g.snapshot.Done && g.accumulator >= g.options.Config.Tick.Seconds(); steps++ {
			g.accumulator -= g.options.Config.Tick.Seconds()
			g.step()
		}
		g.animate(delta)
	}
	return nil
}

func (g *Game) syncDots() {
	targets := make(map[int]point)
	for _, zone := range g.snapshot.Zones {
		for i, id := range zone.StudentIDs {
			targets[id] = slot(zone.Area, i, zone.Capacity)
		}
	}
	// 付款中的学生固定在实际接单窗口，关闭窗口时不会突然换位置。
	for _, worker := range g.snapshot.Workers {
		if worker.Staff.Role == agents.Cashier && worker.StudentID != 0 {
			index := worker.Staff.ID - environment.Servers - 1
			targets[worker.StudentID] = slot(agents.Payment, index, environment.MaxCashiers)
		}
	}
	for _, student := range g.snapshot.Students {
		if student.Status == agents.Scheduled {
			continue
		}
		id := student.Student.ID
		person := g.dots[id]
		if person == nil {
			// 已淡出的学生不再生成；首次出现只发生在入场的时间步。
			if student.Status != agents.Active && student.ExitTime < g.snapshot.Duration {
				continue
			}
			person = &dot{id: id, position: point{970, 835}, preference: student.Student.Preference}
			g.dots[id] = person
		}
		if student.Status == agents.Active {
			target := targets[id]
			if person.area != student.Area {
				person.path = append(person.path, transitionPath(person.area, student.Area, target)...)
			} else if len(person.path) > 0 {
				person.path[len(person.path)-1] = target
			} else {
				person.path = []point{target}
			}
		} else if person.status != student.Status {
			if student.Status == agents.Completed {
				last := person.position
				if len(person.path) > 0 {
					last = person.path[len(person.path)-1]
				}
				person.path = append(person.path, exitPath(id, last)...)
			} else {
				person.path = nil
			}
			person.fade = 3
		}
		person.area, person.status = student.Area, student.Status
	}
}

func (g *Game) animate(delta float64) {
	for id, person := range g.dots {
		distance := 210 * delta
		for len(person.path) > 0 {
			target := person.path[0]
			length := math.Hypot(target.x-person.position.x, target.y-person.position.y)
			if length > distance {
				person.position.x += (target.x - person.position.x) * distance / length
				person.position.y += (target.y - person.position.y) * distance / length
				break
			}
			person.position = target
			person.path = person.path[1:]
			distance -= length
		}
		if person.status != agents.Active && len(person.path) == 0 {
			person.fade -= delta
			if person.fade <= 0 {
				delete(g.dots, id)
			}
		}
	}
}

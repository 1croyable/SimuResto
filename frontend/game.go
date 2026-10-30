package frontend

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"simuresto/simulation"
)

type Options struct {
	Config              simulation.Config
	Students            []simulation.Student
	FontPath            string
	Paused              bool
	Speed               float64
	Screenshot          string
	CaptureAt           time.Duration
	ArrivalInterval     time.Duration
	AutoStart           bool
	PerformanceDuration time.Duration
	PerformanceUncapped bool
	CPUProfile          string
}

type dot struct {
	id         int
	position   point
	path       []point
	area       simulation.Area
	status     simulation.Status
	preference simulation.Preference
	fade       float64
}

type Game struct {
	options        Options
	session        *simulation.Session
	snapshot       simulation.Snapshot
	font           *text.GoTextFaceSource
	chinese        bool
	paused         bool
	speed          float64
	accumulator    float64
	dots           map[int]*dot
	selected       int
	notice         string
	captured       bool
	captureError   error
	viewport       viewport
	configuring    bool
	settingValues  []string
	settingsTab    int
	focusedSetting int
	replaceSetting bool
	settingsError  string
	frame          frameLimiter
	floorLayer     *ebiten.Image
	dotSprites     map[color.NRGBA]*ebiten.Image
	performance    *performanceStats
	renderSize     image.Point
	roundedSprites map[int]*roundedSprite
	circleSprites  map[circleKey]*ebiten.Image
}

func New(options Options) (*Game, error) {
	if options.Config.Capacities.Payment < 1 || options.Config.Capacities.Payment > 2 || options.Config.Cashiers > 2 {
		return nil, fmt.Errorf("图形平面图支持 1–2 个收银位置和收银员")
	}
	if math.IsNaN(options.Speed) || math.IsInf(options.Speed, 0) || options.Speed <= 0 || options.Speed > 32 {
		return nil, fmt.Errorf("播放速度必须在 (0, 32] 之间")
	}
	font, chinese, err := loadFont(options.FontPath)
	if err != nil {
		return nil, err
	}
	game := &Game{options: options, font: font, chinese: chinese, speed: options.Speed}
	game.populateSettings()
	game.configuring = true
	if options.AutoStart {
		if err := game.reset(); err != nil {
			return nil, err
		}
		game.paused = options.Paused
	}
	return game, nil
}

func (g *Game) reset() error {
	g.clearRenderCache()
	if g.session != nil {
		g.session.Close()
	}
	session, err := simulation.NewSession(g.options.Config, g.options.Students)
	if err != nil {
		return err
	}
	g.session = session
	g.snapshot = session.Snapshot()
	g.dots = make(map[int]*dot)
	g.accumulator, g.selected, g.notice = 0, 0, ""
	g.paused = false
	g.configuring = false
	return nil
}

func (g *Game) Close() {
	if g.session != nil {
		g.session.Close()
	}
}

func (g *Game) step() {
	_, err := g.session.Step(context.Background())
	if err != nil && !errors.Is(err, simulation.ErrTimeLimit) {
		g.notice = err.Error()
	}
	if errors.Is(err, simulation.ErrTimeLimit) {
		g.notice = g.label("达到模拟时限", "Time limit reached")
	}
	g.snapshot = g.session.Snapshot()
	g.syncDots()
}

func (g *Game) Update() error {
	if g.performance != nil {
		if g.performance.finished() {
			g.performance.print()
			return ebiten.Termination
		}
		defer g.performance.measureUpdate(time.Now())
	}
	if g.captureError != nil {
		return g.captureError
	}
	if g.captured || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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
		count := 3 - g.options.Config.Capacities.Payment
		g.options.Config.Capacities.Payment, g.options.Config.Cashiers = count, count
		if err := g.reset(); err != nil {
			return err
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
	for _, student := range g.snapshot.Students {
		if student.Status == simulation.Scheduled {
			continue
		}
		id := student.Student.ID
		person := g.dots[id]
		if person == nil {
			// 已淡出的学生不再生成；首次出现只发生在入场的时间步。
			if student.Status != simulation.Active && student.ExitTime < g.snapshot.Duration {
				continue
			}
			person = &dot{id: id, position: point{970, 835}, preference: student.Student.Preference}
			g.dots[id] = person
		}
		if student.Status == simulation.Active {
			target := targets[id]
			if person.area != student.Area {
				person.path = append(person.path, transitionPath(person.area, student.Area, target)...)
			} else if len(person.path) > 0 {
				person.path[len(person.path)-1] = target
			} else {
				person.path = []point{target}
			}
		} else if person.status != student.Status {
			if student.Status == simulation.Completed {
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
		if person.status != simulation.Active && len(person.path) == 0 {
			person.fade -= delta
			if person.fade <= 0 {
				delete(g.dots, id)
			}
		}
	}
}

// 截图由实际 Ebitengine 帧导出，用于保存演示和检查渲染效果。
func (g *Game) capture(screen *ebiten.Image) {
	if g.options.Screenshot == "" || g.captured || (!g.configuring && g.snapshot.Duration < g.options.CaptureAt && !g.snapshot.Done) {
		return
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	pixels := make([]byte, width*height*4)
	screen.ReadPixels(pixels)
	frame := &image.RGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
	file, err := os.Create(g.options.Screenshot)
	if err != nil {
		g.captureError = err
		return
	}
	err = png.Encode(file, frame)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	g.captureError, g.captured = err, err == nil
}

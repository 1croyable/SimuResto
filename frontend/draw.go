package frontend

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"simuresto/simulation"
)

var (
	background = color.NRGBA{242, 244, 247, 255}
	white      = color.NRGBA{255, 255, 255, 255}
	ink        = color.NRGBA{35, 46, 64, 255}
	muted      = color.NRGBA{119, 130, 146, 255}
	border     = color.NRGBA{224, 230, 236, 255}
	orange     = color.NRGBA{230, 124, 56, 255}
	green      = color.NRGBA{37, 156, 128, 255}
	blue       = color.NRGBA{67, 129, 207, 255}
	red        = color.NRGBA{211, 83, 92, 255}
)

func rect(screen *ebiten.Image, r rectangle, fill color.Color) {
	v := screenViewport(screen)
	// 水平、垂直边缘对齐物理像素，无需再执行矢量抗锯齿。
	x, y := math.Round(v.x+float64(r.x)*v.scale), math.Round(v.y+float64(r.y)*v.scale)
	w, h := math.Round(v.x+float64(r.x+r.w)*v.scale)-x, math.Round(v.y+float64(r.y+r.h)*v.scale)-y
	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), fill, false)
}

func line(screen *ebiten.Image, from, to point, width float32, fill color.Color) {
	v := screenViewport(screen)
	vector.StrokeLine(screen, float32(v.x+from.x*v.scale), float32(v.y+from.y*v.scale), float32(v.x+to.x*v.scale), float32(v.y+to.y*v.scale), width*float32(v.scale), fill, true)
}

func arrow(screen *ebiten.Image, from, to point, fill color.Color) {
	line(screen, from, to, 2, fill)
	angle := math.Atan2(to.y-from.y, to.x-from.x)
	for _, offset := range []float64{-0.6, 0.6} {
		line(screen, to, point{to.x - 10*math.Cos(angle+offset), to.y - 10*math.Sin(angle+offset)}, 2, fill)
	}
}

func (g *Game) button(screen *ebiten.Image, r rectangle, label string, active bool) {
	fill, textColor := white, ink
	if active {
		fill, textColor = ink, white
	}
	x, y := g.cursor()
	if r.contains(x, y) && !active {
		fill = color.NRGBA{231, 236, 242, 255}
	}
	g.rounded(screen, r, 9, fill)
	g.text(screen, label, float64(r.x+14), float64(r.y+10), 15, textColor)
}

func (g *Game) Draw(screen *ebiten.Image) {
	var waitStart time.Time
	if g.performance != nil {
		waitStart = time.Now()
	}
	if !g.options.PerformanceUncapped {
		g.frame.wait()
	}
	if g.performance != nil {
		g.performance.measureWait(waitStart)
		g.performance.frame(screen.Bounds().Dx(), screen.Bounds().Dy())
		defer g.performance.measureDraw(time.Now())
	}
	g.prepareRenderCache(screen)
	if g.configuring {
		g.drawSettings(screen)
		g.capture(screen)
		return
	}
	screen.Fill(background)
	g.header(screen)
	g.floorBackground(screen)
	g.zoneBadge(screen, simulation.SelfService, 665, 180)
	g.zoneBadge(screen, simulation.MainService, 274, 180)
	g.people(screen)
	g.sidebar(screen)
	g.text(screen, g.label("空格 暂停/继续    ← → 变速    N 单步    R 重开    P 参数设置    C 切换收银    Esc 退出", "Space Pause    Arrows Speed    N Step    R Restart    P Settings    C Cashiers    Esc Exit"), 26, 873, 13, muted)
	g.text(screen, g.label("按手绘平面图重建 · 圆点位置为移动示意", "Floor plan from sketch · schematic movement"), 1090, 873, 12, muted)
	g.capture(screen)
}

func (g *Game) header(screen *ebiten.Image) {
	g.rounded(screen, rectangle{25, 23, 45, 44}, 12, orange)
	g.text(screen, "S", 37, 24, 30, white)
	g.text(screen, "SimuResto", 84, 16, 29, ink)
	g.text(screen, g.label("CROUS / 实时食堂模拟", "CROUS / LIVE SIMULATION"), 85, 52, 12, muted)
	status, dotColor := g.label("模拟进行中", "RUNNING"), green
	if g.paused {
		status, dotColor = g.label("已暂停", "PAUSED"), orange
	}
	if g.snapshot.Done {
		status, dotColor = g.label("本轮已结束", "FINISHED"), muted
	}
	g.fillCircle(screen, 360, 43, 4, dotColor)
	g.text(screen, status, 372, 31, 15, muted)
	g.drawFPS(screen)
	pauseLabel := g.label("暂停", "Pause")
	if g.paused {
		pauseLabel = g.label("继续", "Resume")
	}
	g.button(screen, pauseButton, pauseLabel, true)
	g.button(screen, speedButton, fmt.Sprintf("%.2gx  +", g.speed), false)
	g.button(screen, restartButton, g.label("参数设置", "Settings"), false)
	g.button(screen, cashierButton, fmt.Sprintf(g.label("收银 %d / 切换重开", "Cashiers %d / reset"), g.options.Config.Cashiers), false)
	g.button(screen, stepButton, g.label("单步", "Step"), false)
}

func (g *Game) panel(screen *ebiten.Image, r rectangle) {
	g.rounded(screen, rectangle{r.x, r.y + 2, r.w, r.h}, 16, color.NRGBA{225, 230, 237, 255})
	g.rounded(screen, r, 16, white)
}

func (g *Game) counter(screen *ebiten.Image, r rectangle, title, subtitle string, tint color.NRGBA) {
	g.rounded(screen, r, 10, tint)
	rect(screen, rectangle{r.x + 12, r.y + r.h - 8, r.w - 24, 3}, color.NRGBA{tint.R - 20, tint.G - 20, tint.B - 20, 255})
	g.text(screen, title, float64(r.x+15), float64(r.y+18), 18, ink)
	g.text(screen, subtitle, float64(r.x+15), float64(r.y+49), 12, muted)
}

func (g *Game) floor(screen *ebiten.Image) {
	g.panel(screen, rectangle{20, 90, 1044, 770})
	g.text(screen, g.label("一层 · 取餐路线", "GROUND FLOOR / SERVICE ROUTE"), 45, 111, 21, ink)
	g.text(screen, g.label("入口 → 自助 → 主菜 → 收银 → 用餐区", "Entrance > Sides > Main > Payment > Dining"), 45, 144, 13, muted)
	g.fillCircle(screen, 680, 124, 5, green)
	g.text(screen, g.label("素食学生", "Vegetarian"), 691, 113, 13, muted)
	g.fillCircle(screen, 800, 124, 5, blue)
	g.text(screen, g.label("肉食倾向", "Meat preference"), 811, 113, 13, muted)
	g.fillCircle(screen, 940, 124, 5, orange)
	g.text(screen, g.label("工作人员", "Staff"), 951, 113, 13, muted)

	// 主路线和收银分流均来自手绘图；用浅橙色标记可行走通道。
	paths := [][]point{
		{{970, 660}, {970, 515}, {820, 515}, {820, 350}, {230, 350}, {140, 345}, {50, 345}, {50, 610}, {660, 610}, {660, 770}},
		{{230, 350}, {230, 390}, {140, 390}, {50, 390}},
	}
	for _, path := range paths {
		for i := 1; i < len(path); i++ {
			line(screen, path[i-1], path[i], 26, color.NRGBA{253, 241, 230, 255})
		}
	}
	for _, pair := range [][2]point{
		{{970, 600}, {970, 565}}, {{920, 515}, {880, 515}}, {{820, 465}, {820, 430}},
		{{770, 350}, {735, 350}}, {{510, 350}, {475, 350}}, {{260, 350}, {235, 350}},
		{{50, 530}, {50, 565}}, {{360, 610}, {410, 610}}, {{660, 640}, {660, 665}},
	} {
		arrow(screen, pair[0], pair[1], orange)
	}

	g.counter(screen, rectangle{65, 220, 125, 85}, g.label("收银 01", "PAY 01"), g.label("付款出口", "CHECKOUT"), color.NRGBA{226, 235, 250, 255})
	second := color.NRGBA{226, 235, 250, 255}
	subtitle := g.label("付款出口", "CHECKOUT")
	if g.options.Config.Cashiers == 1 {
		second, subtitle = color.NRGBA{238, 240, 242, 255}, g.label("暂未开放", "CLOSED")
	}
	g.counter(screen, rectangle{65, 405, 125, 85}, g.label("收银 02", "PAY 02"), subtitle, second)
	g.counter(screen, rectangle{270, 220, 205, 85}, g.label("主菜打饭区", "MAIN COURSES"), g.label("工作人员分餐 · 每人一份", "STAFF SERVICE / ONE MAIN"), color.NRGBA{251, 230, 214, 255})
	g.counter(screen, rectangle{510, 220, 300, 85}, g.label("前菜 · 甜点自取区", "STARTERS / DESSERTS"), g.label("自行选择 · 按库存领取", "SELF SERVICE"), color.NRGBA{221, 240, 232, 255})
	for i := 0; i < 8; i++ {
		g.rounded(screen, rectangle{529 + float32(i)*33, 283, 24, 13}, 3, color.NRGBA{155, 194, 175, 255})
	}
	g.rounded(screen, rectangle{855, 337, 62, 112}, 8, color.NRGBA{235, 238, 243, 255})
	g.text(screen, g.label("餐具", "TRAYS"), 863, 348, 14, ink)
	for i := 0; i < 4; i++ {
		g.rounded(screen, rectangle{870, 382 + float32(i)*7, 31, 5}, 2, white)
	}
	g.counter(screen, rectangle{280, 465, 330, 105}, g.label("饮水 · 调料区", "WATER / CONDIMENTS"), g.label("公共设施 · 暂不计入服务时间", "AMENITY / NO SERVICE DELAY"), color.NRGBA{232, 239, 245, 255})
	for i := 0; i < 3; i++ {
		g.fillCircle(screen, 553+float32(i)*15, 487, 4, blue)
	}
	g.rounded(screen, rectangle{925, 190, 110, 135}, 10, color.NRGBA{246, 247, 249, 255})
	g.text(screen, g.label("通往二楼", "UPSTAIRS"), 937, 205, 15, muted)
	g.text(screen, g.label("暂不模拟", "NOT MODELED"), 938, 231, 11, muted)
	for i := 0; i < 5; i++ {
		line(screen, point{945, 262 + float64(i)*10}, point{1013, 262 + float64(i)*10}, 2, border)
	}

	g.text(screen, g.label("室内排队区", "INDOOR QUEUE"), 861, 555, 16, ink)
	g.text(screen, fmt.Sprintf(g.label("容量 %d 人", "Capacity %d"), g.options.Config.Capacities.Inside), 861, 580, 12, muted)
	line(screen, point{800, 675}, point{935, 675}, 3, border)
	line(screen, point{1005, 675}, point{1040, 675}, 3, border)
	g.text(screen, g.label("入口", "ENTRY"), 946, 650, 13, orange)
	g.rounded(screen, rectangle{827, 692, 214, 146}, 10, color.NRGBA{244, 246, 249, 255})
	g.text(screen, fmt.Sprintf(g.label("室外队列 · %d 人上限", "OUTDOOR / MAX %d"), g.options.Config.Capacities.Outside), 842, 808, 12, muted)

	g.text(screen, g.label("用餐区", "DINING AREA"), 92, 685, 22, ink)
	g.text(screen, g.label("付款后前往此处 · 座位与用餐尚未模拟", "After payment / seating is not modeled yet"), 92, 720, 13, muted)
	for row := 0; row < 2; row++ {
		for column := 0; column < 4; column++ {
			x, y := float32(95+column*118), float32(760+row*45)
			g.rounded(screen, rectangle{x, y, 74, 24}, 5, color.NRGBA{238, 241, 244, 255})
			for _, cx := range []float32{x + 15, x + 57} {
				g.fillCircle(screen, cx, y-5, 3, border)
				g.fillCircle(screen, cx, y+29, 3, border)
			}
		}
	}

	// 空心点标出各区可占用的位置，实心点代表真实学生。
	for _, zone := range g.snapshot.Zones {
		for i := 0; i < zone.Capacity; i++ {
			p := slot(zone.Area, i, zone.Capacity)
			g.strokeCircle(screen, float32(p.x), float32(p.y), 4, 1, border)
		}
	}
}

func (g *Game) zoneBadge(screen *ebiten.Image, area simulation.Area, x, y float32) {
	for _, zone := range g.snapshot.Zones {
		if zone.Area == area {
			width := float32(133)
			if area == simulation.MainService {
				width = 96
			}
			g.rounded(screen, rectangle{x, y, width, 26}, 7, background)
			g.text(screen, fmt.Sprintf(g.label("占用 %d / %d", "Occupied %d / %d"), len(zone.StudentIDs), zone.Capacity), float64(x+10), float64(y+4), 12, muted)
		}
	}
}

func (g *Game) people(screen *ebiten.Image) {
	busy := make(map[int]bool)
	serverIndex, cashierIndex := 0, 0
	for _, worker := range g.snapshot.Workers {
		p := point{398 + float64(serverIndex%3)*30, 180 + float64(serverIndex/3)*18}
		if worker.Staff.Role == simulation.Server {
			serverIndex++
		} else {
			p = point{166, 248 + float64(cashierIndex)*185}
			cashierIndex++
		}
		fill := muted
		if worker.StudentID != 0 {
			fill = orange
			busy[worker.StudentID] = true
		}
		g.fillCircle(screen, float32(p.x), float32(p.y), 8, fill)
		g.fillCircle(screen, float32(p.x), float32(p.y-1), 2, white)
	}
	// 按学生编号绘制，避免 map 遍历顺序造成重叠颜色闪动。
	for _, student := range g.snapshot.Students {
		person := g.dots[student.Student.ID]
		if person == nil {
			continue
		}
		fill := blue
		if person.preference == simulation.Vegetarian {
			fill = green
		}
		if person.status == simulation.Abandoned || person.status == simulation.Rejected || person.status == simulation.Cancelled {
			fill = red
		}
		if person.status != simulation.Active && len(person.path) == 0 {
			fill.A = uint8(max(0, min(255, 255*person.fade/3)))
		}
		x, y := float32(person.position.x), float32(person.position.y)
		g.drawDot(screen, person.position, fill)
		if busy[person.id] {
			g.strokeCircle(screen, x, y, 9, 2, orange)
		}
		if person.id == g.selected {
			g.strokeCircle(screen, x, y, 12, 2, ink)
			g.text(screen, fmt.Sprintf("#%d", person.id), float64(x+13), float64(y-18), 13, ink)
		}
	}
}

func (g *Game) areaName(area simulation.Area) string {
	switch area {
	case simulation.Outside:
		return g.label("室外排队", "Outdoor queue")
	case simulation.Inside:
		return g.label("室内排队", "Indoor queue")
	case simulation.SelfService:
		return g.label("前菜 / 甜点", "Self service")
	case simulation.MainService:
		return g.label("主菜打饭", "Main service")
	case simulation.Payment:
		return g.label("付款", "Payment")
	}
	return g.label("已离场", "Left service")
}

func (g *Game) sidebar(screen *ebiten.Image) {
	g.panel(screen, rectangle{1080, 90, 340, 170})
	g.text(screen, g.label("本轮概况", "LIVE OVERVIEW"), 1100, 106, 16, ink)
	elapsed := g.snapshot.Duration / time.Second
	g.text(screen, fmt.Sprintf("%02d:%02d", elapsed/60, elapsed%60), 1100, 133, 34, ink)
	g.text(screen, g.label("模拟时间", "SIMULATED TIME"), 1242, 152, 12, muted)
	active := 0
	for _, student := range g.snapshot.Students {
		if student.Status == simulation.Active {
			active++
		}
	}
	for i, item := range []struct {
		label string
		value int
	}{
		{g.label("在场", "ACTIVE"), active}, {g.label("完成", "SERVED"), g.snapshot.Metrics.Completed}, {g.label("离队", "LEFT"), g.snapshot.Metrics.Abandoned}, {g.label("满员拒绝", "REJECTED"), g.snapshot.Metrics.Rejected},
	} {
		x := float64(1100 + i*79)
		g.text(screen, fmt.Sprint(item.value), x, 187, 24, ink)
		g.text(screen, item.label, x, 224, 11, muted)
	}
	g.panel(screen, rectangle{1080, 274, 340, 224})
	g.text(screen, g.label("区域占用", "ZONE OCCUPANCY"), 1100, 291, 16, ink)
	for i, zone := range g.snapshot.Zones {
		y := float64(326 + i*31)
		g.text(screen, g.areaName(zone.Area), 1100, y, 13, muted)
		g.rounded(screen, rectangle{1210, float32(y + 6), 122, 7}, 3, background)
		width := float32(len(zone.StudentIDs)) * 122 / float32(zone.Capacity)
		if width > 6 {
			g.rounded(screen, rectangle{1210, float32(y + 6), width, 7}, 3, orange)
		}
		g.text(screen, fmt.Sprintf("%d/%d", len(zone.StudentIDs), zone.Capacity), 1346, y, 13, ink)
	}
	g.panel(screen, rectangle{1080, 512, 340, 204})
	g.inspector(screen)
	g.panel(screen, rectangle{1080, 730, 340, 130})
	g.text(screen, g.label("服务与库存", "SERVICE / STOCK"), 1100, 746, 16, ink)
	remaining := 0
	for _, stock := range g.snapshot.Stock {
		remaining += stock.Remaining
	}
	g.text(screen, fmt.Sprintf(g.label("平均等待   %s", "Average wait   %s"), g.snapshot.Metrics.AverageWait.Round(time.Second)), 1100, 780, 14, muted)
	g.text(screen, fmt.Sprintf(g.label("预备餐品 %d 份 · 兜底主菜不限量", "Prepared %d / fallback unlimited"), remaining), 1100, 807, 13, muted)
	if g.notice != "" {
		g.text(screen, g.fit(g.notice, 12, 300), 1100, 837, 12, red)
	}
}

func (g *Game) inspector(screen *ebiten.Image) {
	var selected *simulation.StudentResult
	for i := range g.snapshot.Students {
		if g.snapshot.Students[i].Student.ID == g.selected {
			selected = &g.snapshot.Students[i]
			break
		}
	}
	if selected == nil {
		g.text(screen, g.label("观察学生", "STUDENT INSPECTOR"), 1100, 530, 16, ink)
		g.text(screen, g.label("点击平面图上的圆点", "Select a dot on the floor plan"), 1100, 573, 15, muted)
		g.text(screen, g.label("查看偏好、满意度与所选主菜", "Inspect preference, mood and meal"), 1100, 601, 13, muted)
		g.text(screen, fmt.Sprintf(g.label("当前平均满意度  %.1f / 100", "Average satisfaction  %.1f / 100"), g.snapshot.Metrics.AverageSatisfaction), 1100, 656, 14, ink)
		return
	}
	g.text(screen, fmt.Sprintf(g.label("学生 #%d", "STUDENT #%d"), selected.Student.ID), 1100, 530, 17, ink)
	preference := g.label("肉食倾向", "Meat preferred")
	if selected.Student.Preference == simulation.Vegetarian {
		preference = g.label("素食", "Vegetarian")
	}
	g.text(screen, preference+" · "+g.areaName(selected.Area), 1100, 560, 13, muted)
	g.text(screen, fmt.Sprintf(g.label("满意度  %.1f / 100", "Satisfaction  %.1f / 100"), selected.Satisfaction), 1100, 589, 14, ink)
	g.rounded(screen, rectangle{1100, 616, 295, 6}, 3, background)
	if selected.Satisfaction > 2 {
		g.rounded(screen, rectangle{1100, 616, 295 * selected.Satisfaction / 100, 6}, 3, green)
	}
	g.text(screen, fmt.Sprintf(g.label("累计等待  %s", "Total wait  %s"), selected.TotalWait.Round(time.Second)), 1100, 634, 13, muted)
	main := g.label("主菜尚未领取", "Main not collected")
	if selected.Meal.Main != nil {
		main = selected.Meal.Main.Name
	}
	g.text(screen, g.fit(main, 14, 295), 1100, 665, 14, ink)
}

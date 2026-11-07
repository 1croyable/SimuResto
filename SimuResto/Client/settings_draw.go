package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) drawSettings(screen *ebiten.Image) {
	screen.Fill(background)
	g.rounded(screen, rectangle{25, 23, 45, 44}, 12, orange)
	g.text(screen, "S", 37, 24, 30, white)
	g.text(screen, "SimuResto", 84, 16, 29, ink)
	g.text(screen, g.label("CROUS / 模拟参数设置", "CROUS / SCENARIO SETUP"), 85, 52, 12, muted)
	g.drawFPS(screen)
	g.text(screen, g.label("准备一个食堂午餐场景", "Build a lunch-service scenario"), 1080, 32, 17, ink)
	g.panel(screen, rectangle{20, 90, 1044, 770})
	g.text(screen, g.label("开始之前", "BEFORE YOU START"), 48, 111, 25, ink)
	g.text(screen, g.label("点击数值直接输入，或用 + / − 调整。开始后才会启动学生和工作人员。", "Type a value or use + / -. Agents start only after you launch the simulation."), 48, 148, 13, muted)
	for i, tab := range settingTabs {
		g.button(screen, settingTabBox(i), g.label(tab[0], tab[1]), g.settingsTab == i)
	}
	if g.settingsTab == preparationTab {
		g.drawPreparation(screen)
	}
	for i := g.settingsTab * 8; i < min(g.settingsTab*8+8, len(settingSpecs)); i++ {
		spec, box := settingSpecs[i], settingBox(i)
		g.text(screen, g.label(spec.cn, spec.en), float64(box.x), float64(box.y-29), 15, ink)
		outline := border
		if i == g.focusedSetting {
			outline = orange
		}
		g.rounded(screen, box, 8, outline)
		fill := white
		if i == g.focusedSetting {
			fill = color.NRGBA{255, 248, 241, 255}
		}
		g.rounded(screen, rectangle{box.x + 1.5, box.y + 1.5, box.w - 3, box.h - 3}, 7, fill)
		value := g.settingValues[i]
		if value == "" && i != g.focusedSetting {
			value = "—"
		}
		if i == g.focusedSetting && g.replaceSetting {
			g.rounded(screen, rectangle{box.x + 10, box.y + 8, min(272, float32(len(value))*13+15), 30}, 3, color.NRGBA{251, 225, 201, 255})
		}
		g.text(screen, value, float64(box.x+15), float64(box.y+9), 20, ink)
		g.text(screen, spec.unit, float64(box.x+box.w-40), float64(box.y+13), 13, muted)
		g.button(screen, rectangle{box.x + 354, box.y, 46, 46}, "−", false)
		g.button(screen, rectangle{box.x + 410, box.y, 46, 46}, "+", false)
		g.text(screen, fmt.Sprintf(g.label("范围 %s–%s", "Range %s–%s"), settingText(spec.minimum), settingText(spec.maximum)), float64(box.x), float64(box.y+52), 12, muted)
	}
	if g.settingsTab == int(mainWeight)/8 {
		g.text(screen, g.label("50 分为中性：高于 50 加分，低于 50 扣分。", "50 is neutral: higher scores add satisfaction; lower scores subtract."), 48, 540, 17, ink)
		g.text(screen, g.label("整餐变化 = 每份菜的（评分 − 50）按权重加权平均。", "Meal change = weighted average of each portion's (score - 50)."), 48, 580, 15, muted)
		g.text(screen, g.label("默认主菜 : 前菜 : 甜点 = 2 : 1 : 1；两份甜点各计算一次权重。", "Default main : starter : dessert = 2 : 1 : 1; each portion counts separately."), 48, 620, 15, muted)
		g.text(screen, g.label("前菜和甜点默认合计两份，可以选 2 + 0、1 + 1 或 0 + 2。", "Two sides by default: 2 starters, one of each, or 2 desserts."), 48, 660, 15, muted)
		g.text(screen, g.label("主菜偏好不符先减 25 分，最低为 0；餐品总影响含缺货最多 ±50 分。", "Mismatched main: score -25, minimum 0. Total food effect including shortages: +/-50."), 48, 700, 15, muted)
		g.text(screen, g.label("温度 0 选最高分；1 为默认；越高越随机。此处统一设置所有学生。", "Temperature: 0 picks the best, 1 is default, higher is more random. Applies to all students."), 48, 740, 13, muted)
	}
	if g.settingsError != "" {
		g.rounded(screen, rectangle{48, 768, 976, 52}, 8, color.NRGBA{253, 235, 235, 255})
		g.text(screen, g.fit(g.settingsError, 15, 942), 64, 781, 15, red)
	} else {
		if g.settingsTab == preparationTab || g.settingsTab == 1 {
			g.text(screen, g.label("主菜份数在备餐页逐项设置；修改素食或荤食总库存，会将该类份数均分。", "Set portions per combination; changing a main stock total redistributes that category evenly."), 48, 781, 14, muted)
		} else {
			g.text(screen, g.label("小提示：到达间隔为 0 可模拟集中到达；室外扣分必须高于室内。", "Tip: arrival interval 0 creates a crowd; outdoor loss must exceed indoor loss."), 48, 781, 14, muted)
		}
	}
	g.panel(screen, rectangle{1080, 90, 340, 770})
	g.text(screen, g.label("场景预览", "SCENARIO PREVIEW"), 1100, 112, 20, ink)
	g.text(screen, g.settingValues[studentCount], 1100, 154, 46, ink)
	g.text(screen, g.label("名学生", "STUDENTS"), 1102, 216, 13, muted)
	g.text(screen, fmt.Sprintf(g.label("每 %s 秒到达一人", "One arrival every %s seconds"), g.settingValues[arrivalSeconds]), 1100, 252, 15, ink)
	g.text(screen, fmt.Sprintf(g.label("%s 名打饭人员 · %s 个收银窗口", "%s servers / %s payment windows"), g.settingValues[serverCount], g.settingValues[cashierCount]), 1100, 286, 14, muted)
	line(screen, point{1100, 326}, point{1400, 326}, 1, border)
	g.text(screen, g.label("服务路线 / 各区容量", "ROUTE / CAPACITY"), 1100, 347, 14, ink)
	for i, row := range []struct {
		cn, en string
		key    settingKey
	}{
		{"室外队列", "Outdoor", outdoorCapacity}, {"室内队列", "Indoor", indoorCapacity}, {"前菜甜点区", "Self service", selfCapacity}, {"主菜区", "Main service", mainCapacity}, {"付款区", "Payment", cashierCount},
	} {
		y := float64(390 + i*38)
		g.fillCircle(screen, 1108, float32(y+10), 4, orange)
		if i < 4 {
			line(screen, point{1108, y + 17}, point{1108, y + 31}, 1.5, border)
		}
		g.text(screen, g.label(row.cn, row.en), 1126, y, 14, muted)
		g.text(screen, g.settingValues[row.key], 1360, y, 15, ink)
	}
	line(screen, point{1100, 590}, point{1400, 590}, 1, border)
	g.text(screen, fmt.Sprintf(g.label("播放速度  %s 倍", "Playback  %sx"), g.settingValues[playbackSpeed]), 1100, 614, 14, ink)
	g.text(screen, fmt.Sprintf(g.label("随机种子  %s", "Seed  %s"), g.settingValues[randomSeed]), 1100, 646, 14, muted)
	g.button(screen, defaultsButton, g.label("恢复默认参数", "Restore defaults"), false)
	g.button(screen, startButton, g.label("开始模拟  →", "Start simulation  >"), true)
	g.text(screen, g.label("Tab 切换输入框    ↑ ↓ 调整数值    Enter 开始模拟    Esc 退出", "Tab Next field    Up / Down Adjust    Enter Start    Esc Exit"), 26, 873, 13, muted)
}

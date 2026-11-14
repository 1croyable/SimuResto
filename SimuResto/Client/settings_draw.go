package client

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	environment "simuresto/SimuResto/Environment"
)

func (g *Game) drawNumberInput(screen *ebiten.Image, box rectangle, value, unit string, focused, replace bool) {
	outline, fill := border, white
	if focused {
		outline, fill = orange, color.NRGBA{255, 248, 241, 255}
	}
	g.rounded(screen, box, 8, outline)
	g.rounded(screen, rectangle{box.x + 1.5, box.y + 1.5, box.w - 3, box.h - 3}, 7, fill)
	if focused && replace && value != "" {
		g.rounded(screen, rectangle{box.x + 8, box.y + 6, min(box.w-16, float32(len(value))*12+8), 30}, 3, color.NRGBA{251, 225, 201, 255})
	}
	g.text(screen, g.fit(value, 20, float64(box.w-24)), float64(box.x+12), float64(box.y+8), 20, ink)
	if unit != "" {
		g.text(screen, unit, float64(box.x+box.w-44), float64(box.y+13), 13, muted)
	}
}

func (g *Game) drawSettings(screen *ebiten.Image) {
	screen.Fill(background)
	g.rounded(screen, rectangle{25, 23, 45, 44}, 12, orange)
	g.text(screen, "S", 37, 24, 30, white)
	g.text(screen, "SimuResto", 84, 16, 29, ink)
	g.text(screen, g.label("CROUS / 开餐准备", "CROUS / BEFORE SERVICE"), 85, 52, 12, muted)
	g.drawFPS(screen)
	g.panel(screen, rectangle{20, 90, 1044, 770})
	g.text(screen, g.label("准备开餐", "PREPARE FOR LUNCH"), 48, 111, 25, ink)
	g.text(screen, g.label("设置人数、窗口和模拟时间，准备好主菜后即可开始。", "Set students, cashiers and time, prepare the mains, then start."), 48, 148, 13, muted)
	for i, tab := range settingTabs {
		g.button(screen, settingTabBox(i), g.label(tab[0], tab[1]), g.settingsTab == i)
	}
	if g.settingsTab == preparationTab {
		g.drawPreparation(screen)
	} else {
		for i, spec := range settingSpecs {
			box := settingBox(i)
			g.text(screen, g.label(spec.cn, spec.en), float64(box.x), float64(box.y-29), 15, ink)
			g.drawNumberInput(screen, box, g.settingValues[i], spec.unit, i == g.focusedSetting, g.replaceSetting)
			g.button(screen, rectangle{box.x + 354, box.y, 46, 46}, "−", false)
			g.button(screen, rectangle{box.x + 410, box.y, 46, 46}, "+", false)
			g.text(screen, fmt.Sprintf(g.label("范围 %s–%s", "Range %s–%s"), settingText(spec.minimum), settingText(spec.maximum)), float64(box.x), float64(box.y+52), 12, muted)
		}
		g.rounded(screen, rectangle{48, 514, 976, 220}, 12, background)
		g.text(screen, g.label("本次模拟使用固定食堂规则", "FIXED CAFETERIA RULES"), 68, 532, 18, ink)
		g.text(screen, g.label("每人 1 份主菜；前菜 + 甜点合计最多 2 份，可自由搭配。", "One main per student; up to two starters / desserts in any combination."), 68, 572, 15, ink)
		g.text(screen, g.label("整餐权重：主菜 : 前菜 : 甜点 = 2 : 1 : 1；50 分为中性。", "Meal weights: main : starter : dessert = 2 : 1 : 1; score 50 is neutral."), 68, 606, 15, muted)
		g.text(screen, g.label("所有菜品合计最多加 / 扣 50 分，学生满意度保持在 0–100。", "Total food effect is limited to +/-50; satisfaction stays within 0–100."), 68, 640, 15, muted)
		g.text(screen, g.label("收银窗口数量与播放速度，可以在模拟运行中随时调整。", "Change the number of open cashiers and playback speed during the simulation."), 68, 683, 15, ink)
	}
	if g.settingsError != "" {
		g.rounded(screen, rectangle{48, 768, 976, 52}, 8, color.NRGBA{253, 235, 235, 255})
		g.text(screen, g.fit(g.settingsError, 15, 942), 64, 781, 15, red)
	} else {
		g.text(screen, g.label("点击数字直接输入，也可以用 + / − 调整；按 Enter 开始。", "Click a number to type, or use + / -. Press Enter to start."), 48, 786, 14, muted)
	}
	g.drawSetupSummary(screen)
	if g.ingredientPicker >= 0 {
		g.drawIngredientPicker(screen)
	}
}

func (g *Game) drawSetupSummary(screen *ebiten.Image) {
	g.panel(screen, rectangle{1080, 90, 340, 770})
	g.text(screen, g.label("开餐概览", "SERVICE OVERVIEW"), 1100, 112, 20, ink)
	g.text(screen, g.settingValues[studentCount], 1100, 154, 46, ink)
	g.text(screen, g.label("名学生", "STUDENTS"), 1102, 216, 13, muted)
	g.text(screen, fmt.Sprintf(g.label("逻辑步长 %s 秒", "Logical step %s s"), g.settingValues[tickSeconds]), 1100, 254, 15, ink)
	g.text(screen, fmt.Sprintf(g.label("最长模拟 %s 分钟", "Duration limit %s min"), g.settingValues[maxMinutes]), 1100, 286, 15, ink)
	line(screen, point{1100, 326}, point{1400, 326}, 1, border)
	total, vegetarian, valid := 0, 0, true
	for _, row := range g.preparation {
		count, err := strconv.Atoi(row.portions)
		if err != nil || count < 0 || count > 10000 {
			valid = false
			continue
		}
		total += count
		if row.dish.Vegetarian {
			vegetarian += count
		}
	}
	g.text(screen, g.label("主菜备餐", "PREPARED MAINS"), 1100, 347, 16, ink)
	if valid {
		g.text(screen, fmt.Sprintf(g.label("%d 份 / %d 种组合", "%d portions / %d combinations"), total, len(g.preparation)), 1100, 381, 20, orange)
		g.text(screen, fmt.Sprintf(g.label("素食 %d 份 · 荤食 %d 份", "Vegetarian %d / meat %d"), vegetarian, total-vegetarian), 1100, 420, 14, muted)
	} else {
		g.text(screen, g.label("请检查备餐份数", "Check the portion counts"), 1100, 381, 18, red)
	}
	g.text(screen, g.label("主菜售罄后自动供应兜底素食。", "Fallback vegetarian mains never run out."), 1100, 458, 13, muted)
	line(screen, point{1100, 500}, point{1400, 500}, 1, border)
	g.text(screen, fmt.Sprintf(g.label("前菜 %d 份 · 甜点 %d 份", "Starters %d / desserts %d"), environment.StarterPortions, environment.DessertPortions), 1100, 523, 15, ink)
	g.text(screen, fmt.Sprintf(g.label("开放收银 %s 个 · 播放 %gx", "%s cashiers open / speed %gx"), g.settingValues[cashierCount], g.speed), 1100, 562, 15, ink)
	g.text(screen, g.label("运行中可切换 1 / 2 个窗口。", "Switch between 1 / 2 cashiers while running."), 1100, 595, 13, muted)
	g.text(screen, g.label("关闭窗口会先完成当前付款。", "Closing a cashier finishes its current payment."), 1100, 624, 13, muted)
	g.button(screen, defaultsButton, g.label("恢复默认设置与菜单", "Restore defaults and menu"), false)
	g.button(screen, startButton, g.label("开始模拟  →", "Start simulation  >"), true)
}

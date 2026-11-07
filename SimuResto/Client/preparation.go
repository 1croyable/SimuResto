package client

import (
	"fmt"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

const preparationTab = 4
const preparationsPerPage = 4

type preparationRow struct {
	dish   agents.Dish
	fields [4]string // 主食、素菜、荤菜评分，以及备餐份数。
}

func newPreparationRow(dish agents.Dish) preparationRow {
	c := dish.Components
	return preparationRow{dish: dish.Prepared(), fields: [4]string{
		settingText(float64(c.Staple.Score)), settingText(float64(c.Vegetable.Score)),
		settingText(float64(c.Protein.Score)), strconv.Itoa(dish.Portions),
	}}
}

func (g *Game) populatePreparation() {
	g.preparation = nil
	for _, dish := range g.options.Config.Menu {
		if dish.Course == agents.Main {
			g.preparation = append(g.preparation, newPreparationRow(dish))
		}
	}
	g.preparationPage, g.preparationFocus = 0, -1
}

func (g *Game) preparedMainDishes() ([]agents.Dish, error) {
	dishes := make([]agents.Dish, 0, len(g.preparation))
	for i, row := range g.preparation {
		dish := row.dish
		count, err := strconv.Atoi(row.fields[3])
		if err != nil || count < 0 || count > 10000 {
			return nil, fmt.Errorf("组合 %d：份数须为 0–10000 的整数", i+1)
		}
		dish.Portions = count
		if dish.Components.IsSet() {
			parts := []*agents.MainIngredient{&dish.Components.Staple, &dish.Components.Vegetable, &dish.Components.Protein}
			for j, part := range parts {
				score, err := strconv.ParseFloat(row.fields[j], 32)
				if err != nil || math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 100 {
					return nil, fmt.Errorf("组合 %d：各食材评分须为 0–100", i+1)
				}
				part.Score = float32(score)
			}
			dish = agents.NewComposedMain(dish.ID, dish.Components, count)
		}
		dishes = append(dishes, dish)
	}
	return dishes, nil
}

// 总量未改变时保留每个组合的备餐数量，用户修改总量才均分。
func applyMainTotals(menu []agents.Dish, vegetarian, meat int) {
	totals := [2]int{}
	for _, dish := range menu {
		if dish.Course == agents.Main {
			index := 1
			if dish.Vegetarian {
				index = 0
			}
			totals[index] += dish.Portions
		}
	}
	if totals[0] != vegetarian {
		distributePortions(menu, agents.Main, true, vegetarian)
	}
	if totals[1] != meat {
		distributePortions(menu, agents.Main, false, meat)
	}
}

func (g *Game) applyPreparationTotals() {
	mains, err := g.preparedMainDishes()
	if err != nil {
		return
	}
	veg, e1 := strconv.Atoi(g.settingValues[vegetarianStock])
	meat, e2 := strconv.Atoi(g.settingValues[meatStock])
	if e1 != nil || e2 != nil || veg < 0 || meat < 0 {
		return
	}
	applyMainTotals(mains, veg, meat)
	for i, dish := range mains {
		g.preparation[i].fields[3] = strconv.Itoa(dish.Portions)
	}
	g.syncPreparationTotals()
}

func (g *Game) syncPreparationTotals() {
	totals := [2]int{}
	for _, row := range g.preparation {
		count, err := strconv.Atoi(row.fields[3])
		if err != nil || count < 0 || count > 10000 {
			return
		}
		index := 1
		if row.dish.Prepared().Vegetarian {
			index = 0
		}
		totals[index] += count
	}
	g.settingValues[vegetarianStock] = strconv.Itoa(totals[0])
	g.settingValues[meatStock] = strconv.Itoa(totals[1])
}

func (g *Game) addPreparation() {
	used := map[string]bool{}
	for _, dish := range g.options.Config.Menu {
		used[dish.ID] = true
	}
	for _, row := range g.preparation {
		used[row.dish.ID] = true
	}
	number := 1
	for used[fmt.Sprintf("prepared_main_%d", number)] {
		number++
	}
	food := environment.DefaultMainIngredients()
	dish := agents.NewComposedMain(fmt.Sprintf("prepared_main_%d", number), agents.MainComponents{
		Staple: food.Staples[0], Vegetable: food.Vegetables[0], Protein: food.Proteins[0],
	}, 20)
	g.preparation = append(g.preparation, newPreparationRow(dish))
	g.preparationPage = (len(g.preparation) - 1) / preparationsPerPage
	g.preparationFocus = -1
	g.syncPreparationTotals()
}

func (g *Game) cycleIngredient(rowIndex, column int) {
	row := &g.preparation[rowIndex]
	food := environment.DefaultMainIngredients()
	choices := [][]agents.MainIngredient{food.Staples, food.Vegetables, food.Proteins}[column]
	if !row.dish.Components.IsSet() {
		row.dish.Components = agents.MainComponents{Staple: food.Staples[0], Vegetable: food.Vegetables[0], Protein: food.Proteins[0]}
		quantity := row.fields[3]
		*row = newPreparationRow(row.dish)
		row.fields[3] = quantity
	}
	parts := []*agents.MainIngredient{&row.dish.Components.Staple, &row.dish.Components.Vegetable, &row.dish.Components.Protein}
	next := 0
	for i, choice := range choices {
		if choice.Name == parts[column].Name {
			next = (i + 1) % len(choices)
			break
		}
	}
	*parts[column] = choices[next]
	row.fields[column] = settingText(float64(choices[next].Score))
	row.dish = row.dish.Prepared()
	g.syncPreparationTotals()
}

func ingredientBox(row, column int) rectangle {
	return rectangle{48 + float32(column)*244, 284 + float32(row%preparationsPerPage)*104, 232, 36}
}

func preparationInputBox(row, column int) rectangle {
	if column == 3 {
		return rectangle{792, 284 + float32(row%preparationsPerPage)*104, 116, 36}
	}
	box := ingredientBox(row, column)
	return rectangle{box.x + 132, box.y + 43, 100, 30}
}

var addPreparationButton = rectangle{48, 714, 220, 38}
var previousPreparationButton = rectangle{714, 714, 130, 38}
var nextPreparationButton = rectangle{894, 714, 130, 38}

func (g *Game) updatePreparation(x, y int) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.preparationFocus = -1
		if addPreparationButton.contains(x, y) {
			g.addPreparation()
			return
		}
		lastPage := max(0, (len(g.preparation)-1)/preparationsPerPage)
		if previousPreparationButton.contains(x, y) {
			g.preparationPage = max(0, g.preparationPage-1)
		}
		if nextPreparationButton.contains(x, y) {
			g.preparationPage = min(lastPage, g.preparationPage+1)
		}
		start := g.preparationPage * preparationsPerPage
		for i := start; i < min(start+preparationsPerPage, len(g.preparation)); i++ {
			for column := 0; column < 3; column++ {
				if ingredientBox(i, column).contains(x, y) {
					g.cycleIngredient(i, column)
				}
			}
			for column := 0; column < 4; column++ {
				if preparationInputBox(i, column).contains(x, y) && (column == 3 || g.preparation[i].dish.Components.IsSet()) {
					g.preparationFocus, g.preparationReplace = i*4+column, true
				}
			}
			if (rectangle{932, ingredientBox(i, 0).y, 92, 36}).contains(x, y) {
				g.preparation = append(g.preparation[:i], g.preparation[i+1:]...)
				g.preparationPage = min(g.preparationPage, max(0, (len(g.preparation)-1)/preparationsPerPage))
				g.syncPreparationTotals()
				return
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if err := g.startConfigured(); err != nil {
			g.settingsError = err.Error()
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && len(g.preparation) > 0 {
		first := g.preparationPage * preparationsPerPage * 4
		end := min(first+preparationsPerPage*4, len(g.preparation)*4)
		g.preparationFocus++
		if g.preparationFocus < first || g.preparationFocus >= end {
			g.preparationFocus = first
		}
		g.preparationReplace = true
	}
	if g.preparationFocus < 0 {
		return
	}
	row, column := g.preparationFocus/4, g.preparationFocus%4
	if row >= len(g.preparation) {
		return
	}
	value := &g.preparation[row].fields[column]
	previousValue := *value
	if (ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.preparationReplace = true
	}
	for _, char := range ebiten.AppendInputChars(nil) {
		if (char >= '0' && char <= '9') || (char == '.' && column != 3) {
			if g.preparationReplace {
				*value = ""
				g.preparationReplace = false
			}
			if len(*value) < 8 {
				*value += string(char)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if g.preparationReplace {
			*value = ""
			g.preparationReplace = false
		} else if len(*value) > 0 {
			*value = (*value)[:len(*value)-1]
		}
	}
	adjustment := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		adjustment = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		adjustment = -1
	}
	if adjustment != 0 {
		current, _ := strconv.ParseFloat(*value, 64)
		limit := 100.0
		if column == 3 {
			limit = 10000
		}
		*value = settingText(min(limit, max(0, current+float64(adjustment))))
		g.preparationReplace = false
	}
	if *value != previousValue {
		g.syncPreparationTotals()
		g.settingsError = ""
	}
}

func (g *Game) drawPreparation(screen *ebiten.Image) {
	g.text(screen, g.label("点击食材切换，点击分数或份数编辑。植物肉计为素食。", "Click ingredients to cycle; edit scores and portions. Plant proteins are vegetarian."), 48, 234, 13, muted)
	for i, title := range []string{g.label("主食 ×2", "Starch x2"), g.label("素菜 ×1", "Vegetables x1"), g.label("荤菜 / 植物肉 ×3", "Protein x3")} {
		g.text(screen, title, float64(48+i*244), 260, 14, ink)
	}
	g.text(screen, g.label("备餐份数", "Portions"), 792, 260, 14, ink)
	mains, err := g.preparedMainDishes()
	start := g.preparationPage * preparationsPerPage
	for i := start; i < min(start+preparationsPerPage, len(g.preparation)); i++ {
		row := g.preparation[i]
		c := row.dish.Components
		parts := []agents.MainIngredient{c.Staple, c.Vegetable, c.Protein}
		for column, part := range parts {
			box := ingredientBox(i, column)
			name := part.Name
			if name == "" {
				name = g.label("选择食材", "Choose ingredient")
			}
			g.button(screen, box, g.fit(name+"  >", 14, 210), false)
			g.text(screen, g.label("食材评分", "Score"), float64(box.x), float64(box.y+48), 12, muted)
		}
		for column := 0; column < 4; column++ {
			box := preparationInputBox(i, column)
			outline := border
			if g.preparationFocus == i*4+column {
				outline = orange
			}
			g.rounded(screen, box, 5, outline)
			g.rounded(screen, rectangle{box.x + 1, box.y + 1, box.w - 2, box.h - 2}, 4, white)
			g.text(screen, row.fields[column], float64(box.x+10), float64(box.y+4), 17, ink)
		}
		y := ingredientBox(i, 0).y
		g.button(screen, rectangle{932, y, 92, 36}, g.label("删除", "Delete"), false)
		if err == nil {
			kind := g.label("荤", "Meat")
			if mains[i].Vegetarian {
				kind = g.label("素", "Veg")
			}
			g.text(screen, fmt.Sprintf("%s · %.1f / 100", kind, mains[i].SatisfactionScore), 792, float64(y+48), 13, ink)
		}
	}
	if len(g.preparation) == 0 {
		g.text(screen, g.label("暂无预制主菜，运行时将使用兜底素食。", "No prepared mains; fallback vegetarian meal will be used."), 48, 310, 16, muted)
	}
	g.button(screen, addPreparationButton, g.label("+ 添加主菜组合", "+ Add combination"), false)
	g.text(screen, fmt.Sprintf(g.label("%d 种组合 · 基准 50 分", "%d combinations / baseline 50"), len(g.preparation)), 292, 723, 13, muted)
	g.button(screen, previousPreparationButton, g.label("上一页", "Previous"), false)
	g.button(screen, nextPreparationButton, g.label("下一页", "Next"), false)
	g.text(screen, strconv.Itoa(g.preparationPage+1), 861, 723, 14, ink)
}

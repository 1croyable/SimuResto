package client

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

const preparationTab = 1
const preparationsPerPage = 4

type preparationRow struct {
	dish     agents.Dish
	portions string // 输入中可能暂时为空，开始模拟前再校验。
}

func newPreparationRow(dish agents.Dish) preparationRow {
	return preparationRow{dish: dish.Prepared(), portions: strconv.Itoa(dish.Portions)}
}

func (g *Game) populatePreparation() {
	g.preparation = nil
	for _, dish := range g.options.Config.Menu {
		if dish.Course == agents.Main {
			g.preparation = append(g.preparation, newPreparationRow(dish))
		}
	}
	g.preparationPage, g.preparationFocus, g.ingredientPicker = 0, -1, -1
}

func (g *Game) preparedMainDishes() ([]agents.Dish, error) {
	dishes := make([]agents.Dish, 0, len(g.preparation))
	for i, row := range g.preparation {
		count, err := strconv.Atoi(row.portions)
		if err != nil || count < 0 || count > 10000 {
			g.preparationPage, g.preparationFocus = i/preparationsPerPage, i
			return nil, fmt.Errorf(g.label("组合 %d：份数须为 0–10000 的整数", "Combination %d: enter 0–10000 portions"), i+1)
		}
		dish := row.dish.Prepared()
		dish.Portions = count
		dishes = append(dishes, dish)
	}
	return dishes, nil
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
	g.preparationFocus, g.ingredientPicker = len(g.preparation)-1, -1
	g.preparationReplace = true
	g.settingsError = ""
}

func ingredientChoices(column int) []agents.MainIngredient {
	food := environment.DefaultMainIngredients()
	return [][]agents.MainIngredient{food.Staples, food.Vegetables, food.Proteins}[column]
}

func (g *Game) selectIngredient(rowIndex, column, choice int) {
	row := &g.preparation[rowIndex]
	if !row.dish.Components.IsSet() {
		food := environment.DefaultMainIngredients()
		row.dish.Components = agents.MainComponents{Staple: food.Staples[0], Vegetable: food.Vegetables[0], Protein: food.Proteins[0]}
	}
	parts := []*agents.MainIngredient{&row.dish.Components.Staple, &row.dish.Components.Vegetable, &row.dish.Components.Protein}
	*parts[column] = ingredientChoices(column)[choice]
	row.dish = agents.NewComposedMain(row.dish.ID, row.dish.Components, row.dish.Portions)
	g.ingredientPicker, g.settingsError = -1, ""
}

func ingredientBox(row, column int) rectangle {
	return rectangle{48 + float32(column)*236, 300 + float32(row%preparationsPerPage)*100, 218, 42}
}

func preparationInputBox(row int) rectangle {
	return rectangle{800, ingredientBox(row, 0).y, 84, 42}
}

var addPreparationButton = rectangle{48, 714, 220, 38}
var previousPreparationButton = rectangle{714, 714, 130, 38}
var nextPreparationButton = rectangle{894, 714, 130, 38}

func (g *Game) updatePreparation(x, y int, clicked bool) {
	if clicked {
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
					g.ingredientPicker = i*3 + column
					return
				}
			}
			box := preparationInputBox(i)
			if box.contains(x, y) {
				g.preparationFocus, g.preparationReplace = i, true
			}
			if (rectangle{760, box.y, 34, 42}).contains(x, y) {
				adjustNumber(&g.preparation[i].portions, 0, 10000, -1)
				g.settingsError = ""
			}
			if (rectangle{890, box.y, 34, 42}).contains(x, y) {
				adjustNumber(&g.preparation[i].portions, 0, 10000, 1)
				g.settingsError = ""
			}
			if (rectangle{940, box.y, 84, 42}).contains(x, y) {
				g.preparation = append(g.preparation[:i], g.preparation[i+1:]...)
				g.preparationPage = min(g.preparationPage, max(0, (len(g.preparation)-1)/preparationsPerPage))
				g.settingsError = ""
				return
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && len(g.preparation) > 0 {
		first := g.preparationPage * preparationsPerPage
		end := min(first+preparationsPerPage, len(g.preparation))
		g.preparationFocus++
		if g.preparationFocus < first || g.preparationFocus >= end {
			g.preparationFocus = first
		}
		g.preparationReplace = true
	}
	if g.preparationFocus < 0 {
		return
	}
	value := &g.preparation[g.preparationFocus].portions
	before := *value
	editNumber(value, &g.preparationReplace, true)
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		adjustNumber(value, 0, 10000, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		adjustNumber(value, 0, 10000, -1)
	}
	if before != *value {
		g.settingsError = ""
	}
}

// 列表靠近页面底部时向上展开，保证所有食材都可见。
func (g *Game) ingredientPickerBox() rectangle {
	row, column := g.ingredientPicker/3, g.ingredientPicker%3
	box := ingredientBox(row, column)
	height := float32(len(ingredientChoices(column))*44 + 8)
	y := box.y + box.h + 4
	if y+height > 706 {
		y = box.y - height - 4
	}
	return rectangle{box.x, y, 242, height}
}

func (g *Game) chooseIngredient(x, y int) {
	row, column := g.ingredientPicker/3, g.ingredientPicker%3
	box := g.ingredientPickerBox()
	for i := range ingredientChoices(column) {
		if (rectangle{box.x + 4, box.y + 4 + float32(i)*44, box.w - 8, 44}).contains(x, y) {
			g.selectIngredient(row, column, i)
			return
		}
	}
	g.ingredientPicker = -1
}

func (g *Game) drawPreparation(screen *ebiten.Image) {
	g.text(screen, g.label("选择三种食材，再填写份数。食材评分固定，组合评分自动计算。", "Choose three ingredients and set portions. Scores are fixed and combined automatically."), 48, 238, 14, muted)
	for i, title := range []string{g.label("主食 ×2", "Starch x2"), g.label("素菜 ×1", "Vegetables x1"), g.label("荤菜 / 植物肉 ×3", "Protein x3")} {
		g.text(screen, title, float64(48+i*236), 270, 14, ink)
	}
	g.text(screen, g.label("备餐份数", "Portions"), 800, 270, 14, ink)
	start := g.preparationPage * preparationsPerPage
	for i := start; i < min(start+preparationsPerPage, len(g.preparation)); i++ {
		row := g.preparation[i]
		c := row.dish.Components
		for column, part := range []agents.MainIngredient{c.Staple, c.Vegetable, c.Protein} {
			box := ingredientBox(i, column)
			name := part.Name
			if name == "" {
				name = g.label("选择食材", "Choose ingredient")
			}
			g.rounded(screen, box, 8, background)
			g.text(screen, g.fit(name, 15, 174), float64(box.x+12), float64(box.y+10), 15, ink)
			g.text(screen, "v", float64(box.x+box.w-22), float64(box.y+10), 14, muted)
			g.text(screen, fmt.Sprintf(g.label("评分 %g / 100", "Score %g / 100"), part.Score), float64(box.x+12), float64(box.y+49), 12, muted)
		}
		box := preparationInputBox(i)
		g.drawNumberInput(screen, box, row.portions, "", g.preparationFocus == i, g.preparationReplace)
		g.button(screen, rectangle{760, box.y, 34, 42}, "−", false)
		g.button(screen, rectangle{890, box.y, 34, 42}, "+", false)
		g.button(screen, rectangle{940, box.y, 84, 42}, g.label("删除", "Delete"), false)
		kind, tint := g.label("荤食", "Meat"), blue
		if row.dish.Vegetarian {
			kind, tint = g.label("素食", "Veg"), green
		}
		g.text(screen, fmt.Sprintf(g.label("%s · 综合 %.1f 分", "%s / score %.1f"), kind, row.dish.SatisfactionScore), 768, float64(box.y+49), 13, tint)
		line(screen, point{48, float64(box.y + 82)}, point{1024, float64(box.y + 82)}, 1, border)
	}
	if len(g.preparation) == 0 {
		g.text(screen, g.label("还没有主菜组合。点击下方按钮添加。", "No combinations yet. Add one below."), 48, 340, 18, ink)
		g.text(screen, g.label("也可以直接开始，食堂会供应不限量的兜底素食。", "You can also start with unlimited fallback vegetarian mains."), 48, 378, 14, muted)
	}
	g.button(screen, addPreparationButton, g.label("+ 添加主菜组合", "+ Add combination"), false)
	g.text(screen, fmt.Sprintf(g.label("%d 种组合 · 植物肉归为素食", "%d combinations / plant proteins are vegetarian"), len(g.preparation)), 288, 725, 13, muted)
	g.button(screen, previousPreparationButton, g.label("上一页", "Previous"), false)
	g.button(screen, nextPreparationButton, g.label("下一页", "Next"), false)
	g.text(screen, strconv.Itoa(g.preparationPage+1), 861, 725, 14, ink)
}

func (g *Game) drawIngredientPicker(screen *ebiten.Image) {
	row, column := g.ingredientPicker/3, g.ingredientPicker%3
	box := g.ingredientPickerBox()
	g.panel(screen, box)
	c := g.preparation[row].dish.Components
	current := []agents.MainIngredient{c.Staple, c.Vegetable, c.Protein}[column]
	x, y := g.cursor()
	for i, choice := range ingredientChoices(column) {
		item := rectangle{box.x + 4, box.y + 4 + float32(i)*44, box.w - 8, 44}
		if item.contains(x, y) || current.Name == choice.Name {
			g.rounded(screen, item, 6, background)
		}
		g.text(screen, g.fit(choice.Name, 14, 160), float64(item.x+10), float64(item.y+11), 14, ink)
		g.text(screen, fmt.Sprintf("%g", choice.Score), float64(item.x+item.w-42), float64(item.y+11), 14, green)
	}
}

package agents

import "strings"

// MainIngredient 表示一份主菜中的食材，植物肉的 ContainsMeat 为 false。
type MainIngredient struct {
	Name         string
	Score        float32
	ContainsMeat bool
}

// MainComponents 按主食、素菜、荤菜（含植物肉）三个位置拼装主菜。
// 使用值类型，菜单、餐盘与快照之间复制时不会共享可修改的配方。
type MainComponents struct {
	Staple    MainIngredient
	Vegetable MainIngredient
	Protein   MainIngredient
}

func (c MainComponents) IsSet() bool { return c != (MainComponents{}) }

// Score 以 50 为基准，三部分偏离基准的分数按 2:1:3 加权。
func (c MainComponents) Score() float32 {
	return float32(50 + (2*(float64(c.Staple.Score)-50)+(float64(c.Vegetable.Score)-50)+3*(float64(c.Protein.Score)-50))/6)
}

func (c MainComponents) Vegetarian() bool {
	return !c.Staple.ContainsMeat && !c.Vegetable.ContainsMeat && !c.Protein.ContainsMeat
}

// Prepared 由组合重新计算主菜分数和饮食分类，避免使用过期的缓存字段。
func (dish Dish) Prepared() Dish {
	if dish.Course == Main && dish.Components.IsSet() {
		dish.SatisfactionScore = dish.Components.Score()
		dish.Vegetarian = dish.Components.Vegetarian()
	}
	return dish
}

func NewComposedMain(id string, components MainComponents, portions int) Dish {
	return (Dish{
		ID:         id,
		Name:       strings.Join([]string{components.Staple.Name, components.Vegetable.Name, components.Protein.Name}, " + "),
		Course:     Main,
		Portions:   portions,
		Components: components,
	}).Prepared()
}

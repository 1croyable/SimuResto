package simulation

// 食堂库存管理结构体
type inventory struct {
	items        []Stock        //库存列表，保存每种菜的剩余份数、预留份数、已领取份数
	byID         map[string]int //菜品到item中的索引
	fallbackMain int            // 评分最低的素食主菜，常规候选售罄后自动补充。
}

func newInventory(menu []Dish) *inventory {
	stock := &inventory{byID: make(map[string]int), fallbackMain: -1}
	for _, dish := range menu {
		stock.byID[dish.ID] = len(stock.items)
		stock.items = append(stock.items, Stock{Dish: dish, Remaining: dish.Portions})
		if dish.Course == Main && dish.Vegetarian &&
			(stock.fallbackMain < 0 || dish.SatisfactionScore < stock.items[stock.fallbackMain].Dish.SatisfactionScore) {
			stock.fallbackMain = len(stock.items) - 1
		}
	}
	// 自定义菜单没有素食主菜时，仍保证有一份 0 分的备用主菜。
	if stock.fallbackMain < 0 {
		id := "fallback_vegetarian"
		for {
			if _, exists := stock.byID[id]; !exists {
				break
			}
			id += "_"
		}
		dish := Dish{
			ID:                id,
			Name:              "备用清煮蔬菜",
			Course:            Main,
			Vegetarian:        true,
			Portions:          0,
			SatisfactionScore: 0,
		}
		stock.fallbackMain = len(stock.items)
		stock.byID[id] = stock.fallbackMain
		stock.items = append(stock.items, Stock{Dish: dish})
	}
	return stock
}

func (inventory *inventory) takeSides(choices []string, count int) []Dish {
	var dishes []Dish
	for range count {
		found := false
		for _, id := range choices {
			stock := &inventory.items[inventory.byID[id]]
			if stock.Remaining > 0 {
				stock.Remaining--
				stock.Taken++
				dishes = append(dishes, stock.Dish)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return dishes
}

// reserveMain 优先消耗常规库存，全部不可用时改用兜底素食；返回值标记是否启用兜底。
func (inventory *inventory) reserveMain(choices []string) (string, bool) {
	for _, id := range choices {
		stock := &inventory.items[inventory.byID[id]]
		if stock.Remaining > 0 {
			stock.Remaining--
			stock.Reserved++
			return id, false
		}
	}
	stock := &inventory.items[inventory.fallbackMain]
	if stock.Remaining == 0 {
		stock.Remaining++
		stock.Replenished++
	}
	stock.Remaining--
	stock.Reserved++
	return stock.Dish.ID, true
}

// deliverMain 将一份预留库存转换为一份已交付主菜。
func (stock *inventory) deliverMain(id string) Dish {
	item := &stock.items[stock.byID[id]]
	item.Reserved--
	item.Taken++
	return item.Dish
}

func (stock *inventory) releaseMain(id string) {
	item := &stock.items[stock.byID[id]]
	item.Reserved--
	item.Remaining++
}

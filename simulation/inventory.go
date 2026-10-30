package simulation

// 食堂库存管理结构体
type inventory struct {
	items []Stock        //库存列表，保存每种菜的剩余份数、预留份数、已领取份数
	byID  map[string]int //菜品到item中的索引
}

func newInventory(menu []Dish) *inventory {
	stock := &inventory{byID: make(map[string]int)}
	for _, dish := range menu {
		stock.byID[dish.ID] = len(stock.items)
		stock.items = append(stock.items, Stock{Dish: dish, Remaining: dish.Portions})
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

func (inventory *inventory) reserveMain(choices []string) string {
	for _, id := range choices {
		stock := &inventory.items[inventory.byID[id]]
		if stock.Remaining > 0 {
			stock.Remaining--
			stock.Reserved++
			return id
		}
	}
	return ""
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

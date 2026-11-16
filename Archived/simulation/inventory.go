package simulation

import "sync"

// 食堂库存管理结构体
type inventory struct {
	mu           sync.Mutex     // 保护完整的选菜及库存变更过程。
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

// availableDishes 只返回当前有库存的菜品；肉食主菜有货时优先提供肉类。
// 调用方必须持有 mu，直到选中的份数被扣除或预留。
func (inventory *inventory) availableDishes(course Course, preference Preference) []Dish {
	var dishes, meatMains []Dish
	for _, stock := range inventory.items {
		dish := stock.Dish
		if stock.Remaining <= 0 || dish.Course != course || (preference == Vegetarian && !dish.Vegetarian) {
			continue
		}
		dishes = append(dishes, dish)
		if course == Main && !dish.Vegetarian {
			meatMains = append(meatMains, dish)
		}
	}
	if preference == MeatPreferred && len(meatMains) > 0 {
		return meatMains
	}
	return dishes
}

func (inventory *inventory) takeSide(id string) Dish {
	stock := &inventory.items[inventory.byID[id]]
	stock.Remaining--
	stock.Taken++
	return stock.Dish
}

// reserveMain 优先消耗常规库存，全部不可用时改用兜底素食；返回值标记是否启用兜底。
// 与 takeSide 一样，调用方必须持有 mu。
func (inventory *inventory) reserveMain(id string) (string, bool) {
	if id != "" {
		stock := &inventory.items[inventory.byID[id]]
		stock.Remaining--
		stock.Reserved++
		return id, false
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
	stock.mu.Lock()
	defer stock.mu.Unlock()
	item := &stock.items[stock.byID[id]]
	item.Reserved--
	item.Taken++
	return item.Dish
}

func (stock *inventory) releaseMain(id string) {
	stock.mu.Lock()
	defer stock.mu.Unlock()
	item := &stock.items[stock.byID[id]]
	item.Reserved--
	item.Remaining++
}

package environment

import (
	"fmt"
	"time"

	agents "simuresto/SimuResto/Agents"
)

const (
	Starter1Counter = iota
	Starter2Counter
	MainCounter
)

// CounterStock is the on-display stock of one dish.
// Each counter is represented by a slice of CounterStock.
type CounterStock struct {
	DishID   string
	Portions int
}

type RestockState struct {
	Active       bool
	CounterIndex int
	DishID       string
	Remaining    time.Duration
}

func (w *Environment) counterIndex(d agents.Dish) int {
	if d.Course == agents.Main {
		return MainCounter
	}
	if d.StarterCounter == 1 {
		return Starter1Counter
	}
	return Starter2Counter
}

// InitCounters builds the three counter inventories from Menu.
// No dish starts with displayed stock; only per-dish MaxPortions limits apply.
func (w *Environment) InitCounters() {
	if w.Settings.RestockDuration <= 0 {
		panic("restock duration must be positive")
	}
	w.Counters = [3][]CounterStock{}

	for i := range w.Menu {
		dish := &w.Menu[i]
		if dish.RefillBatch <= 0 || dish.MaxPortions <= 0 {
			panic(fmt.Sprintf("invalid refill batch or maximum portions for %s", dish.ID))
		}
		if dish.Course != agents.Main && (dish.Course != agents.Starter ||
			(dish.StarterCounter != 1 && dish.StarterCounter != 2)) {
			panic(fmt.Sprintf("invalid counter for %s", dish.ID))
		}

		idx := w.counterIndex(*dish)
		dish.Portions = 0
		w.Counters[idx] = append(w.Counters[idx], CounterStock{DishID: dish.ID})
	}
}

func (w *Environment) findDish(id string) *agents.Dish {
	for i := range w.Menu {
		if w.Menu[i].ID == id {
			return &w.Menu[i]
		}
	}
	return nil
}

func (w *Environment) counterStock(id string) (int, *CounterStock) {
	for i := range w.Counters {
		for j := range w.Counters[i] {
			if w.Counters[i][j].DishID == id {
				return i, &w.Counters[i][j]
			}
		}
	}
	return -1, nil
}

func (w *Environment) Stock(id string) int {
	_, item := w.counterStock(id)
	if item == nil {
		return 0
	}
	return item.Portions
}

func (w *Environment) HasStock(id string, needed int) bool {
	return w.Stock(id) >= needed
}

// RefillOptions lists dishes that are completely sold out.
// Once selected, the cook keeps restocking the same dish to MaxPortions.
func (w *Environment) RefillOptions() []agents.RestockOption {
	var options []agents.RestockOption

	for i := range w.Counters {
		for _, item := range w.Counters[i] {
			if item.Portions != 0 {
				continue
			}
			dish := w.findDish(item.DishID)
			if dish == nil || dish.RefillBatch <= 0 || dish.MaxPortions <= 0 {
				continue
			}
			options = append(options, agents.RestockOption{
				CounterIndex: i,
				DishID:       item.DishID,
			})
		}
	}
	return options
}

// RefillDish completes one restocking batch. The final batch is truncated
// to MaxPortions and still takes one full restocking duration.
func (w *Environment) RefillDish(counterIndex int, dishID string) bool {
	if counterIndex < 0 || counterIndex >= len(w.Counters) {
		return false
	}
	dish := w.findDish(dishID)
	idx, item := w.counterStock(dishID)
	if dish == nil || item == nil || idx != counterIndex || dish.RefillBatch <= 0 {
		return false
	}
	remaining := dish.MaxPortions - item.Portions
	if remaining <= 0 {
		return false
	}

	batch := min(dish.RefillBatch, remaining)
	item.Portions += batch
	dish.Portions = item.Portions
	return true
}

func (w *Environment) AtMaxStock(dishID string) bool {
	dish := w.findDish(dishID)
	return dish != nil && w.Stock(dishID) >= dish.MaxPortions
}

func (w *Environment) TakeDish(id string) bool {
	dish := w.findDish(id)
	_, item := w.counterStock(id)
	if dish == nil || item == nil || item.Portions <= 0 {
		return false
	}
	item.Portions--
	dish.Portions = item.Portions
	return true
}

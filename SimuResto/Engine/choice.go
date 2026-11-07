package engine

import (
	"math"
	"math/rand"

	agents "simuresto/SimuResto/Agents"
)

// weightedChoice 从当前可用的菜品中抽取一份，返回菜品 ID。
// 将原权重转成 logit = ln(1 + 个人评分)，再使用 softmax(logit / 温度)。
// 温度为 1 时保持原有概率，为 0 时选择最高分，同分按当前菜单顺序选择。
func weightedChoice(rng *rand.Rand, dishes []agents.Dish, preference agents.Preference, temperature float32) string {
	// probs = torch.softmax(logits / temperature)
	if len(dishes) == 0 {
		return ""
	}
	best := 0
	logits := make([]float64, len(dishes))
	for i, dish := range dishes {
		// logit = ln(1 + score)
		logits[i] = math.Log1p(float64(dishScoreForStudent(dish, preference)))
		if logits[i] > logits[best] {
			best = i
		}
	}
	if temperature == 0 {
		return dishes[best].ID
	}

	// Softmax: ensure value stable!!
	weights := make([]float64, len(dishes))
	var total float64
	for i, logit := range logits {
		weights[i] = math.Exp((logit - logits[best]) / float64(temperature))
		total += weights[i]
	}

	// next_token = torch.multinomial(probs, num_samples=1)
	draw := rng.Float64() * total
	for i, dish := range dishes {
		draw -= weights[i]
		if draw < 0 {
			return dish.ID
		}
	}
	return dishes[len(dishes)-1].ID // err
}

// chooseDish 在实际领取时把当前可选菜品发给学生，由其 goroutine 抽取一份。
// 协调器收到回复后立即扣库存，中间不处理其他学生的取餐请求。
// 调用方必须持有库存锁，覆盖筛选、学生抽选和后续扣库存的全过程。
func (e *engine) chooseDish(student *studentState, course agents.Course) string {
	dishes := e.stock.availableDishes(course, student.result.Student.Preference)
	if len(dishes) == 0 {
		return ""
	}
	student.agent.in <- observation{area: student.result.Area, dishes: dishes}
	return (<-student.agent.out).dishID
}

// reserveStudentMain 将筛选、抽选、补货或预留作为一个不可分割的操作。
func (e *engine) reserveStudentMain(student *studentState) (string, bool) {
	e.stock.mu.Lock()
	defer e.stock.mu.Unlock()
	return e.stock.reserveMain(e.chooseDish(student, agents.Main))
}

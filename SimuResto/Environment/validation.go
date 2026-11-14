package environment

import (
	"fmt"
	"math"

	agents "simuresto/SimuResto/Agents"
)

// Validate 检查模拟配置、菜单及学生资料是否有效。
func (c Config) Validate(students []agents.Student) error {
	if c.Tick <= 0 || c.MaxDuration <= 0 {
		return fmt.Errorf("逻辑步长和最长模拟时间必须大于零")
	}
	if c.Cashiers < 1 || c.Cashiers > MaxCashiers {
		return fmt.Errorf("收银窗口数量只能为 1 或 2")
	}
	dishIDs := make(map[string]bool)
	for _, dish := range c.Menu {
		if err := validateMainComponents(dish); err != nil {
			return err
		}
		dish = dish.Prepared()
		if dish.ID == "" || dishIDs[dish.ID] {
			return fmt.Errorf("dish IDs must be nonempty and unique: %q", dish.ID)
		}
		dishIDs[dish.ID] = true
		if dish.Course != agents.Starter && dish.Course != agents.Main && dish.Course != agents.Dessert {
			return fmt.Errorf("invalid course for dish %q", dish.ID)
		}
		if dish.Portions < 0 {
			return fmt.Errorf("negative portions for dish %q", dish.ID)
		}
		if math.IsNaN(float64(dish.SatisfactionScore)) || math.IsInf(float64(dish.SatisfactionScore), 0) ||
			dish.SatisfactionScore < 0 || dish.SatisfactionScore > 100 {
			return fmt.Errorf("dish satisfaction score must be between 0 and 100: %q", dish.ID)
		}
	}
	studentIDs := make(map[int]bool)
	for _, student := range students {
		if student.ID <= 0 || studentIDs[student.ID] {
			return fmt.Errorf("student IDs must be positive and unique: %d", student.ID)
		}
		studentIDs[student.ID] = true
		if math.IsNaN(float64(student.Temperature)) || math.IsInf(float64(student.Temperature), 0) || student.Temperature < 0 {
			return fmt.Errorf("student temperature must be finite and nonnegative: %d", student.ID)
		}
		if student.Preference != agents.Vegetarian && student.Preference != agents.MeatPreferred {
			return fmt.Errorf("invalid preference for student %d", student.ID)
		}
		if student.ArrivalTime < 0 || student.Patience < 0 || student.StartersWanted < 0 || student.DessertsWanted < 0 {
			return fmt.Errorf("negative student attributes for student %d", student.ID)
		}
		if math.IsNaN(float64(student.Satisfaction)) || math.IsInf(float64(student.Satisfaction), 0) ||
			student.Satisfaction < 0 || student.Satisfaction > 100 {
			return fmt.Errorf("satisfaction must be between 0 and 100 for student %d", student.ID)
		}
	}
	return nil
}

package environment

import (
	"fmt"
	"math"
	"time"

	agents "simuresto/SimuResto/Agents"
)

// Validate 检查模拟配置、菜单及学生资料是否有效。
func (c Config) Validate(students []agents.Student) error {
	for _, item := range []struct {
		name  string
		value int
	}{
		{"outside capacity", c.Capacities.Outside}, {"inside capacity", c.Capacities.Inside},
		{"self-service capacity", c.Capacities.SelfService}, {"main-service capacity", c.Capacities.MainService},
		{"payment capacity", c.Capacities.Payment}, {"servers", c.Servers}, {"cashiers", c.Cashiers},
	} {
		if item.value <= 0 {
			return fmt.Errorf("%s must be positive", item.name)
		}
	}
	for _, item := range []struct {
		name  string
		value time.Duration
	}{
		{"tick", c.Tick}, {"maximum duration", c.MaxDuration}, {"self-service time", c.SelfServiceTime},
		{"main-service time", c.MainServiceTime}, {"payment time", c.PaymentTime},
	} {
		if item.value <= 0 {
			return fmt.Errorf("%s must be positive", item.name)
		}
	}
	if c.TickInterval < 0 {
		return fmt.Errorf("tick interval cannot be negative")
	}
	if c.MaxStartersPerStudent < 0 || c.MaxDessertsPerStudent < 0 || c.MaxSelfServiceItems < 0 {
		return fmt.Errorf("self-service limits cannot be negative")
	}
	for _, value := range []float32{
		c.IndoorLossPerSecond, c.OutdoorLossPerSecond, c.MissingSidePenalty,
	} {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < 0 {
			return fmt.Errorf("satisfaction adjustments must be finite and nonnegative")
		}
	}
	if c.OutdoorLossPerSecond <= c.IndoorLossPerSecond {
		return fmt.Errorf("outdoor satisfaction loss must be greater than indoor loss")
	}
	weights := c.SatisfactionWeights
	for _, weight := range []float32{weights.Main, weights.Starter, weights.Dessert} {
		if math.IsNaN(float64(weight)) || math.IsInf(float64(weight), 0) || weight < 0 {
			return fmt.Errorf("satisfaction weights must be finite and nonnegative")
		}
	}
	if weights == (CourseWeights{}) {
		return fmt.Errorf("at least one satisfaction weight must be positive")
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

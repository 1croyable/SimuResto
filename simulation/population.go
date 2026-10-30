package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// GenerateStudents 使用独立随机源生成可复现的学生属性。
func GenerateStudents(count int, seed int64, arrivalInterval time.Duration) ([]Student, error) {
	if count < 0 || arrivalInterval < 0 {
		return nil, fmt.Errorf("student count and arrival interval cannot be negative")
	}
	if count > 1 && int64(arrivalInterval) > math.MaxInt64/int64(count-1) {
		return nil, fmt.Errorf("arrival schedule overflows time.Duration")
	}
	rng := rand.New(rand.NewSource(seed))
	students := make([]Student, count)
	for i := range students {
		preference := MeatPreferred
		if rng.Float64() < 0.35 {
			preference = Vegetarian
		}
		students[i] = Student{
			ID: i + 1, Preference: preference, ArrivalTime: time.Duration(i) * arrivalInterval,
			Patience:     time.Duration(40+rng.Intn(121)) * time.Second,
			Satisfaction: float32(80 + rng.Intn(16)), StartersWanted: 1, DessertsWanted: 1,
		}
	}
	return students, nil
}

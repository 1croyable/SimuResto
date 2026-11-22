package environment

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Default configuration values and simulation constants.
const (
	DefaultCashiers = 2
	MaxCashiers     = 2
	SaucePickupTime = 6 * time.Second
)

type Settings struct {
	StartTime time.Duration // Simulation start time.

	// Students
	StudentCount int
	Seed         int64
	BasePatience time.Duration

	// Movement and queues
	QueueSpacing float64
	WalkingSpeed float64 // Map units per simulated second.

	// Simulation timing and rendering
	FramesPerSecond int
	TicksPerSecond  int

	// Staff and service
	Cashiers        int
	ServiceTime     time.Duration
	RestockDuration time.Duration

	PaymentTime time.Duration

	ChoiceTemperature float64 // food selection

	// Dining and sauce station
	MinDiningTime time.Duration
	MaxDiningTime time.Duration
	SaucePortions int
	SauceCapacity int // length of queue of sauce

	Lunch       ArrivalPeriod
	Dinner      ArrivalPeriod
	DinnerShare float64 // Fraction of StudentCount assigned to dinner.
}

type ArrivalPeriod struct {
	Open, Close, Peak     time.Duration
	EarlyLead             time.Duration
	SigmaLeft, SigmaRight time.Duration
	EarlyRatio            float64
	UniformRatio          float64
	StopBeforeClose       time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		StartTime: 11 * time.Hour,

		StudentCount: 1000,
		Seed:         42,
		BasePatience: 10 * time.Minute,

		QueueSpacing: 16,
		WalkingSpeed: 180,

		FramesPerSecond: 60,
		TicksPerSecond:  60,

		Cashiers:        DefaultCashiers,
		ServiceTime:     2 * time.Second,
		RestockDuration: 10 * time.Second,

		PaymentTime: 4 * time.Second,

		ChoiceTemperature: 0.3,
		MinDiningTime:     8 * time.Minute,
		MaxDiningTime:     20 * time.Minute,
		SaucePortions:     100,
		SauceCapacity:     8,

		Lunch: ArrivalPeriod{
			Open:            11*time.Hour + 30*time.Minute, // 11:30
			Close:           13*time.Hour + 45*time.Minute, // 13:45
			Peak:            12*time.Hour + 10*time.Minute, // 12:10
			EarlyLead:       15 * time.Minute,
			SigmaLeft:       20 * time.Minute,
			SigmaRight:      35 * time.Minute,
			EarlyRatio:      0.02,
			UniformRatio:    0.20,
			StopBeforeClose: 10 * time.Minute,
		},

		Dinner: ArrivalPeriod{
			Open:            18*time.Hour + 15*time.Minute, // 18:15
			Close:           20*time.Hour + 30*time.Minute, // 20:30
			Peak:            18*time.Hour + 30*time.Minute, // 18:30
			EarlyLead:       10 * time.Minute,
			SigmaLeft:       20 * time.Minute,
			SigmaRight:      30 * time.Minute,
			EarlyRatio:      0.02,
			UniformRatio:    0.20,
			StopBeforeClose: 10 * time.Minute,
		},

		DinnerShare: 0.35,
	}
}

// used for UI
type SettingField struct {
	Key   string
	Label string
	Value string
	Hint  string
}

var durationType = reflect.TypeOf(time.Duration(0))

func (s Settings) Fields() []SettingField {
	fields := make([]SettingField, 0, 40)
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, prefix string) {
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			spec := typ.Field(i)

			if !spec.IsExported() {
				continue
			}

			value := v.Field(i)
			key := spec.Name

			if prefix != "" {
				key = prefix + "." + key
			}

			if value.Kind() == reflect.Struct {
				walk(value, key)
				continue
			}
			entry := SettingField{Key: key, Label: prettySettingName(key)}

			switch {
			case value.Type() == durationType:
				if clockSetting(key) {
					d := time.Duration(value.Int())
					entry.Value = fmt.Sprintf("%02d:%02d", int(d/time.Hour), int((d%time.Hour)/time.Minute))
					entry.Hint = "HH:MM"
				} else {
					entry.Value = time.Duration(value.Int()).String()
					entry.Hint = "duration: 10m / 2s"
				}
			case value.Kind() == reflect.Int, value.Kind() == reflect.Int64:
				entry.Value = strconv.FormatInt(value.Int(), 10)
				entry.Hint = "integer"
			case value.Kind() == reflect.Float64:
				entry.Value = strconv.FormatFloat(value.Float(), 'f', -1, 64)
				entry.Hint = "decimal"
			default:
				continue
			}

			fields = append(fields, entry)
		}
	}

	walk(reflect.ValueOf(s), "")
	return fields
}

func (s *Settings) SetField(key, input string) error {
	value := reflect.ValueOf(s).Elem()
	for _, component := range strings.Split(key, ".") {
		if value.Kind() != reflect.Struct {
			return fmt.Errorf("unknown setting %q", key)
		}
		value = value.FieldByName(component)
		if !value.IsValid() || !value.CanSet() {
			return fmt.Errorf("unknown setting %q", key)
		}
	}

	input = strings.TrimSpace(input)
	var err error
	switch {
	case value.Type() == durationType:
		var duration time.Duration
		if clockSetting(key) {
			duration, err = parseSettingClock(input)
		} else {
			duration, err = time.ParseDuration(input)
		}
		if err == nil {
			value.SetInt(int64(duration))
		}
	case value.Kind() == reflect.Int, value.Kind() == reflect.Int64:
		var n int64
		n, err = strconv.ParseInt(input, 10, 64)
		if err == nil && value.OverflowInt(n) {
			err = fmt.Errorf("integer out of range")
		}
		if err == nil {
			value.SetInt(n)
		}
	case value.Kind() == reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(input, 64)
		if err == nil && (math.IsNaN(f) || math.IsInf(f, 0)) {
			err = fmt.Errorf("finite number required")
		}
		if err == nil {
			value.SetFloat(f)
		}
	default:
		err = fmt.Errorf("unsupported setting type")
	}
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func clockSetting(key string) bool {
	return key == "StartTime" || key == "Lunch.Open" ||
		key == "Lunch.Close" || key == "Lunch.Peak" ||
		key == "Dinner.Open" || key == "Dinner.Close" || key == "Dinner.Peak"
}

func parseSettingClock(input string) (time.Duration, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("use HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute")
	}
	if hour < 0 || hour >= 24 || minute < 0 || minute >= 60 {
		return 0, fmt.Errorf("time must be between 00:00 and 23:59")
	}
	return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute, nil
}

func prettySettingName(key string) string {
	parts := strings.Split(key, ".")
	for i, part := range parts {
		var b strings.Builder
		var previous rune
		for j, r := range part {
			if j > 0 && unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous)) {
				b.WriteRune(' ')
			}
			b.WriteRune(r)
			previous = r
		}
		parts[i] = b.String()
	}
	return strings.Join(parts, " / ")
}

// checks simulation invariants before creating Environment.
func (s Settings) Validate() error {
	positive := func(name string, v time.Duration) error {
		if v <= 0 {
			return fmt.Errorf("%s must be positive", name)
		}
		return nil
	}
	positiveFloat := func(name string, v float64) error {
		if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%s must be a positive finite number", name)
		}
		return nil
	}
	if s.StartTime < 0 || s.StartTime >= 24*time.Hour {
		return fmt.Errorf("Start Time must be between 00:00 and 23:59")
	}
	if s.StudentCount <= 0 {
		return fmt.Errorf("Student Count must be positive")
	}
	if err := positive("Base Patience", s.BasePatience); err != nil {
		return err
	}
	if err := positiveFloat("Queue Spacing", s.QueueSpacing); err != nil {
		return err
	}
	if err := positiveFloat("Walking Speed", s.WalkingSpeed); err != nil {
		return err
	}
	if s.FramesPerSecond <= 0 || s.TicksPerSecond <= 0 {
		return fmt.Errorf("FPS and TPS must be positive")
	}
	if s.Cashiers < 1 || s.Cashiers > MaxCashiers {
		return fmt.Errorf("Cashiers must be between 1 and %d", MaxCashiers)
	}
	for _, item := range []struct {
		name     string
		duration time.Duration
	}{
		{"Service Time", s.ServiceTime}, {"Restock Duration", s.RestockDuration},
		{"Payment Time", s.PaymentTime}, {"Min Dining Time", s.MinDiningTime},
	} {
		if err := positive(item.name, item.duration); err != nil {
			return err
		}
	}
	if s.MaxDiningTime < s.MinDiningTime {
		return fmt.Errorf("Max Dining Time must be at least Min Dining Time")
	}
	if s.SaucePortions < 0 {
		return fmt.Errorf("Sauce Portions must be non-negative")
	}
	if s.SauceCapacity <= 0 {
		return fmt.Errorf("Sauce Capacity must be positive")
	}
	if s.ChoiceTemperature < 0 || math.IsNaN(s.ChoiceTemperature) || math.IsInf(s.ChoiceTemperature, 0) {
		return fmt.Errorf("Choice Temperature must be a non-negative finite number")
	}
	if s.DinnerShare < 0 || s.DinnerShare > 1 || math.IsNaN(s.DinnerShare) {
		return fmt.Errorf("Dinner Share must be between 0 and 1")
	}
	if err := s.Lunch.validate("Lunch"); err != nil {
		return err
	}
	if err := s.Dinner.validate("Dinner"); err != nil {
		return err
	}
	if s.Lunch.Close > s.Dinner.Open-s.Dinner.EarlyLead {
		return fmt.Errorf("Lunch and Dinner arrival windows must not overlap")
	}
	return nil
}

func (p ArrivalPeriod) validate(name string) error {
	cutoff := p.Close - p.StopBeforeClose
	if p.Open < 0 || p.Close > 24*time.Hour || p.Open >= p.Close ||
		p.EarlyLead <= 0 || p.EarlyLead > p.Open ||
		p.StopBeforeClose < 0 || cutoff <= p.Open ||
		p.Peak <= p.Open || p.Peak >= cutoff ||
		p.SigmaLeft <= 0 || p.SigmaRight <= 0 ||
		math.IsNaN(p.EarlyRatio) || math.IsNaN(p.UniformRatio) ||
		p.EarlyRatio < 0 || p.EarlyRatio > 1 ||
		p.UniformRatio < 0 || p.UniformRatio > 1 {
		return fmt.Errorf("%s: invalid arrival period parameters", name)
	}
	return nil
}

package engine

import (
	"math"
	"testing"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func testWorld() *environment.Environment {
	settings := environment.DefaultSettings()
	settings.WalkingSpeed = 100
	world := environment.New(settings)
	world.Route = []agents.Point{{0, 0}, {100, 0}, {100, 100}}
	world.Student.Position = world.Route[0]
	return world
}

func TestMovementAndServingUseTheSameTimeBudget(t *testing.T) {
	world := testWorld()
	Step(world, 2500*time.Millisecond)
	if world.Student.Position != world.Route[2] || world.Student.Status != agents.Serving {
		t.Fatal("student should reach the counter after two simulated seconds")
	}
	if world.Cook.RemainingService != 1500*time.Millisecond || world.Student.Meal.Main != nil {
		t.Fatal("only the half second left after walking should count toward service")
	}
	Step(world, 1500*time.Millisecond)
	if world.Student.Status != agents.Served || world.Student.Meal.Main == nil || world.RemainingPortions != 0 {
		t.Fatal("completed service should transfer exactly one dish")
	}
	Step(world, time.Hour)
	if world.RemainingPortions != 0 {
		t.Fatal("a served student must not take more dishes")
	}
}

func TestStepSizeDoesNotChangeProgress(t *testing.T) {
	large, small := testWorld(), testWorld()
	Step(large, 3500*time.Millisecond)
	for range 350 {
		Step(small, 10*time.Millisecond)
	}
	if math.Hypot(large.Student.Position.X-small.Student.Position.X,
		large.Student.Position.Y-small.Student.Position.Y) > 1e-6 ||
		large.Student.Status != small.Student.Status || large.Cook.RemainingService != small.Cook.RemainingService {
		t.Fatal("splitting simulated time into smaller steps should preserve progress")
	}
}

func TestPauseAndArrival(t *testing.T) {
	world := testWorld()
	world.Paused = true
	Step(world, time.Second)
	if world.Elapsed != 0 || world.Student.Position != world.Route[0] {
		t.Fatal("pause must freeze the simulation clock and movement")
	}
	world.Paused = false
	world.Student.Student.ArrivalTime = time.Second
	Step(world, 1500*time.Millisecond)
	if world.Student.Position.X != 50 {
		t.Fatal("movement must count only the time after arrival")
	}
	before := world.Elapsed
	Step(world, -time.Second)
	if world.Elapsed != before {
		t.Fatal("negative steps must not reverse time")
	}
}

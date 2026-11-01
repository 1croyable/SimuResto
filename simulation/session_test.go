package simulation

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSessionMatchesContinuousRun(t *testing.T) {
	config := DefaultConfig()
	students, err := GenerateStudents(90, config.Seed, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Run(context.Background(), config, students)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(config, students)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for {
		done, err := session.Step(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
	}
	if !reflect.DeepEqual(want, session.Report()) {
		t.Fatal("图形前端逐步推进与命令行连续运行必须一致")
	}
	if !session.Snapshot().Done {
		t.Fatal("完成后必须标记结束")
	}
}

func TestSnapshotIsIndependentAndReadingDoesNotAdvanceTime(t *testing.T) {
	config := DefaultConfig()
	student := Student{ID: 1, Preference: Vegetarian, Satisfaction: 50, Patience: time.Hour, StartersWanted: 1, DessertsWanted: 1}
	session, err := NewSession(config, []Student{student})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for range 12 {
		if _, err := session.Step(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	want := session.Snapshot()
	changed := session.Snapshot()
	if changed.Students[0].Meal.Main == nil {
		t.Fatal("本场景应已交付主菜")
	}
	changed.Students[0].Meal.Main.Name = "被外部修改"
	changed.Students[0].Meal.Starters[0].Name = "被外部修改"
	changed.Students[0].Waiting[Inside] = time.Hour
	changed.Students[0].Path[0] = Outside
	changed.Stock[0].Remaining = -100
	changed.PeakOccupancy[Inside] = 999
	changed.Metrics.AverageWaitByArea[Inside] = time.Hour
	for i := range changed.Zones {
		if len(changed.Zones[i].StudentIDs) > 0 {
			changed.Zones[i].StudentIDs[0] = 999
		}
	}
	for i := range changed.Events {
		if len(changed.Events[i].DishIDs) > 0 {
			changed.Events[i].DishIDs[0] = "被外部修改"
		}
	}
	if !reflect.DeepEqual(want, session.Snapshot()) {
		t.Fatal("读取或修改快照不应推进时间或改变模拟内部状态")
	}
}

func TestSessionCloseReturnsReservationAndIsIdempotent(t *testing.T) {
	config := DefaultConfig()
	student := Student{ID: 1, Preference: Vegetarian, Satisfaction: 50, Patience: time.Hour}
	session, err := NewSession(config, []Student{student})
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		session.Step(context.Background())
	}
	reserved := 0
	for _, stock := range session.Snapshot().Stock {
		reserved += stock.Reserved
	}
	if reserved != 1 {
		t.Fatal("关闭前应有一份预留主菜")
	}
	session.Close()
	session.Close()
	for _, stock := range session.Snapshot().Stock {
		if stock.Reserved != 0 || stock.Remaining+stock.Taken != stock.Dish.Portions+stock.Replenished {
			t.Fatal("关闭窗口必须释放预留库存")
		}
	}
	if done, err := session.Step(context.Background()); !done || !errors.Is(err, context.Canceled) {
		t.Fatal("已关闭的会话不得再次推进")
	}
}

package engine

import (
	agents "simuresto/SimuResto/Agents"
)

// ZoneSnapshot 保留队首到队尾的顺序，图形前端据此摆放学生。
type ZoneSnapshot struct {
	Area       agents.Area
	Capacity   int
	StudentIDs []int
}

type WorkerSnapshot struct {
	Staff     agents.Staff
	StudentID int  // 0 表示当前空闲。
	Open      bool // 是否接受新任务；关闭时仍可能在完成最后一笔付款。
}

// Snapshot 是当前时间点的独立副本，不包含 channel 或引擎内部指针。
type Snapshot struct {
	Report
	Zones   []ZoneSnapshot
	Workers []WorkerSnapshot
	Done    bool
}

func (g *Game) Snapshot() Snapshot {
	snapshot := Snapshot{Report: g.Report(), Done: g.done}
	for _, area := range agents.ServiceRoute() {
		queue := g.engine.queues[area]
		snapshot.Zones = append(snapshot.Zones, ZoneSnapshot{
			Area: area, Capacity: queue.Capacity(), StudentIDs: queue.IDs(),
		})
	}
	for _, worker := range g.engine.staff {
		view := WorkerSnapshot{Staff: worker.result.Staff, Open: g.engine.workerOpen(worker)}
		if worker.job != nil {
			view.StudentID = worker.job.studentID
		}
		snapshot.Workers = append(snapshot.Workers, view)
	}
	return snapshot
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	copy := make(map[K]V, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func cloneReport(source Report) Report {
	result := source
	result.Students = append([]StudentResult(nil), source.Students...)
	for i := range result.Students {
		student := &result.Students[i]
		student.Waiting = cloneMap(student.Waiting)
		student.Path = append([]agents.Area(nil), student.Path...)
		student.Meal.Starters = append([]agents.Dish(nil), student.Meal.Starters...)
		student.Meal.Desserts = append([]agents.Dish(nil), student.Meal.Desserts...)
		if student.Meal.Main != nil {
			main := *student.Meal.Main
			student.Meal.Main = &main
		}
	}
	result.Staff = append([]StaffResult(nil), source.Staff...)
	result.Stock = append([]agents.Stock(nil), source.Stock...)
	result.Events = append([]Event(nil), source.Events...)
	for i := range result.Events {
		result.Events[i].DishIDs = append([]string(nil), result.Events[i].DishIDs...)
	}
	result.PeakOccupancy = cloneMap(source.PeakOccupancy)
	result.Metrics.AverageWaitByArea = cloneMap(source.Metrics.AverageWaitByArea)
	return result
}

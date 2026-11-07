package engine

import (
	"time"

	agents "simuresto/SimuResto/Agents"
)

//食堂模拟的事件日志，何时发生了什么

type Event struct {
	Time      time.Duration
	Signal    agents.Signal
	StudentID int
	StaffID   int
	From      agents.Area
	To        agents.Area
	DishIDs   []string
}

func (e *engine) emit(signal agents.Signal, id, staffID int, from, to agents.Area, dishes ...string) {
	e.stats.events = append(e.stats.events, Event{
		Time: e.now, Signal: signal, StudentID: id, StaffID: staffID,
		From: from, To: to, DishIDs: dishes,
	})
}

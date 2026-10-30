package simulation

//食堂模拟的事件日志，何时发生了什么
import "time"

type Event struct {
	Time      time.Duration
	Signal    Signal
	StudentID int
	StaffID   int
	From      Area
	To        Area
	DishIDs   []string
}

func (e *engine) emit(signal Signal, id, staffID int, from, to Area, dishes ...string) {
	e.events = append(e.events, Event{
		Time: e.now, Signal: signal, StudentID: id, StaffID: staffID,
		From: from, To: to, DishIDs: dishes,
	})
}

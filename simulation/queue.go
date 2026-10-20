package simulation

import "fmt"

// Queue is a bounded FIFO. Its owner must serialize access; the simulation
// coordinator owns all queues, so agents never mutate them concurrently.
type Queue struct {
	capacity int
	ids      []int
}

func NewQueue(capacity int) (*Queue, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("queue capacity must be positive")
	}
	return &Queue{capacity: capacity}, nil
}

func (q *Queue) Len() int      { return len(q.ids) }
func (q *Queue) Capacity() int { return q.capacity }
func (q *Queue) IDs() []int    { return append([]int(nil), q.ids...) }

func (q *Queue) Enqueue(id int) bool {
	if q.Len() == q.capacity {
		return false
	}
	q.ids = append(q.ids, id)
	return true
}

func (q *Queue) Front() (int, bool) {
	if q.Len() == 0 {
		return 0, false
	}
	return q.ids[0], true
}

func (q *Queue) Dequeue() (int, bool) {
	id, ok := q.Front()
	if ok {
		q.ids = q.ids[1:]
	}
	return id, ok
}

func (q *Queue) Remove(id int) bool {
	for i, candidate := range q.ids {
		if candidate == id {
			copy(q.ids[i:], q.ids[i+1:])
			q.ids = q.ids[:len(q.ids)-1]
			return true
		}
	}
	return false
}

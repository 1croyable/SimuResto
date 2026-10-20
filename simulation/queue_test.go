package simulation

import (
	"reflect"
	"testing"
)

func TestBoundedFIFOAndDeparture(t *testing.T) {
	queue, err := NewQueue(3)
	if err != nil {
		t.Fatal(err)
	}
	if !queue.Enqueue(1) || !queue.Enqueue(2) || !queue.Enqueue(3) || queue.Enqueue(4) {
		t.Fatal("capacity enforcement failed")
	}
	snapshot := queue.IDs()
	snapshot[0] = 99
	if !queue.Remove(2) || queue.Remove(4) {
		t.Fatal("removing a waiting student failed")
	}
	if !queue.Enqueue(4) || !reflect.DeepEqual(queue.IDs(), []int{1, 3, 4}) {
		t.Fatal("departure changed FIFO order")
	}
	for _, want := range []int{1, 3, 4} {
		if id, ok := queue.Dequeue(); !ok || id != want {
			t.Fatalf("got %d, %v; want %d", id, ok, want)
		}
	}
	if _, ok := queue.Dequeue(); ok {
		t.Fatal("dequeued an empty queue")
	}
	if _, err := NewQueue(0); err == nil {
		t.Fatal("zero capacity accepted")
	}
}

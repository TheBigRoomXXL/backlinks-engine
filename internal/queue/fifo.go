package queue

import (
	"container/list"
	"net/url"
	"sync"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
)

// FIFOQueue represents a thread-safe in-memory FIFO queue implemented using a linked list
// and a map for link deduplication.
type FIFOQueue struct {
	mu   *sync.Mutex
	list *list.List
	seen map[string]struct{}
}

func NewFIFOQueue() *FIFOQueue {
	return &FIFOQueue{
		mu:   &sync.Mutex{},
		list: list.New(),
		seen: make(map[string]struct{}),
	}
}

func (q *FIFOQueue) Add(url *url.URL) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, exist := q.seen[url.String()]
	if !exist {
		q.seen[url.String()] = struct{}{}
		q.list.PushBack(url)
		telemetry.QueueSize.Add(1)
	}
	return nil
}

// Next removes and returns the element from the front of the list.
// If the queue is empty, it returns nil.
func (q *FIFOQueue) Next() (*url.URL, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	element := q.list.Front()
	if element != nil {
		q.list.Remove(element)
		telemetry.QueueSize.Add(-1)
		return element.Value.(*url.URL), nil // Type assertion
	}
	return nil, nil
}

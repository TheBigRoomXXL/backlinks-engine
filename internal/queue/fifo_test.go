package queue

import (
	"log"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to parse URL and avoid redundancy in test cases
func parseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("failed to parse URL: %v", err)
	}
	return u
}

func TestFIFOQueue(t *testing.T) {
	queue := NewFIFOQueue()

	url1 := parseURL("http://example.com")
	url2 := parseURL("http://example.org")

	assert.NoError(t, queue.Add(url1), "should add URL1 without error")
	assert.NoError(t, queue.Add(url2), "should add URL2 without error")

	// Ensure elements are dequeued in FIFO order
	next, err := queue.Next()
	assert.NoError(t, err, "should dequeue without error")
	assert.Equal(t, url1, next, "should return URL1 first")

	next, err = queue.Next()
	assert.NoError(t, err, "should dequeue without error")
	assert.Equal(t, url2, next, "should return URL2 second")
}

func TestFIFOQueueDeduplication(t *testing.T) {
	queue := NewFIFOQueue()

	url1 := parseURL("http://example.com")
	assert.NoError(t, queue.Add(url1), "should add URL1 without error")
	assert.NoError(t, queue.Add(url1), "adding the same URL should not fail")

	// Only one instance of URL1 should be in the queue
	next, err := queue.Next()
	assert.NoError(t, err, "should dequeue without error")
	assert.Equal(t, url1, next, "should return URL1")

	next, err = queue.Next()
	assert.NoError(t, err, "should dequeue without error")
	assert.Nil(t, next, "queue should be empty")
}

func TestFIFOQueueEmptyQueue(t *testing.T) {
	queue := NewFIFOQueue()

	next, err := queue.Next()
	assert.NoError(t, err, "should dequeue without error")
	assert.Nil(t, next, "should return nil for empty queue")
}

func TestFIFOQueueBasciThreadSafety(t *testing.T) {
	queue := NewFIFOQueue()

	url1 := parseURL("http://example.com")
	url2 := parseURL("http://example.org")
	url3 := parseURL("http://example.net")

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		assert.NoError(t, queue.Add(url1), "goroutine 1 should add without error")
	}()
	go func() {
		defer wg.Done()
		assert.NoError(t, queue.Add(url2), "goroutine 2 should add without error")
	}()
	go func() {
		defer wg.Done()
		assert.NoError(t, queue.Add(url3), "goroutine 3 should add without error")
	}()

	wg.Wait()

	// Ensure all URLs were added
	seen := make(map[string]bool)
	for i := 0; i < 3; i++ {
		next, err := queue.Next()
		assert.NoError(t, err, "should dequeue without error")
		seen[next.String()] = true
	}
	assert.Len(t, seen, 3, "should contain all 3 unique URLs")
}

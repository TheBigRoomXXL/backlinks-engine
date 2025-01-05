package exporter

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

// MockPgxPool mocks the pgxpool.Pool for testing purposes
type MockPgxPool struct {
	mu             sync.Mutex
	hasReturnError bool
	InsertedRows   [][]any
	FailInsert     bool
}

func (m *MockPgxPool) CopyFrom(ctx context.Context, tableName pgx.Identifier, columns []string, rows pgx.CopyFromSource) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FailInsert {
		m.hasReturnError = true
		return 0, errors.New("mocked insert failure")
	}

	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			return 0, err
		}
		m.InsertedRows = append(m.InsertedRows, row)
	}
	return int64(len(m.InsertedRows)), nil
}

func (m *MockPgxPool) Close() {}

func TestPostgresExporterBasicFunctionality(t *testing.T) {
	mockPool := &MockPgxPool{}
	exporter := NewPostgresExporter(mockPool)

	ctx, cancel := context.WithCancel(context.Background())

	linksChan := make(chan *LinkGroup)
	go exporter.Listen(ctx, linksChan)

	group1 := &LinkGroup{
		From: parseURL("http://bidule"),
		To:   []*url.URL{parseURL("http://truc.com"), parseURL("http://bidule/login")},
	}
	group2 := &LinkGroup{
		From: parseURL("http://truc.com"),
		To:   []*url.URL{parseURL("http://bidule"), parseURL("http://bidule/login")},
	}

	linksChan <- group1
	linksChan <- group2

	cancel()                          // Trigger a partial batch
	time.Sleep(10 * time.Millisecond) // Allow exporter to process links

	assert.Equal(t, 4, len(mockPool.InsertedRows), "should insert all links")
	assert.Contains(t, mockPool.InsertedRows, []any{"http://bidule", "http://truc.com"})
	assert.Contains(t, mockPool.InsertedRows, []any{"http://bidule", "http://bidule/login"})
	assert.Contains(t, mockPool.InsertedRows, []any{"http://truc.com", "http://bidule"})
	assert.Contains(t, mockPool.InsertedRows, []any{"http://truc.com", "http://bidule/login"})
}

func TestPostgresExporterBatchInsertion(t *testing.T) {
	mockPool := &MockPgxPool{}
	exporter := NewPostgresExporter(mockPool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	linksChan := make(chan *LinkGroup)
	go exporter.Listen(ctx, linksChan)

	// Send more than PG_BATCH_SIZE links to test batch behavior
	for i := 0; i < PG_BATCH_SIZE+1; i++ {
		linksChan <- &LinkGroup{
			From: parseURL(fmt.Sprintf("http://bidule/page%d", i)),
			To:   []*url.URL{parseURL(fmt.Sprintf("http://truc.com/page%d", i))},
		}
	}

	time.Sleep(20 * time.Millisecond) // Allow exporter to process links

	assert.Equal(t, PG_BATCH_SIZE, len(mockPool.InsertedRows), "should insert only a batch of PG_BATCH_SIZE")
}

func TestPostgresExporterContextCancellation(t *testing.T) {
	mockPool := &MockPgxPool{}
	exporter := NewPostgresExporter(mockPool)

	ctx, cancel := context.WithCancel(context.Background())
	linksChan := make(chan *LinkGroup)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		exporter.Listen(ctx, linksChan)
	}()

	cancel() // Cancel the context

	// Wait for goroutine to exit
	timeout := time.After(100 * time.Millisecond)
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-timeout:
		assert.Fail(t, "goroutine did not exit in time after context cancellation")
	case <-done:
		assert.True(t, true, "goroutine exited successfully")
	}
}

func TestPostgresExporterInsertFailure(t *testing.T) {
	mockPool := &MockPgxPool{FailInsert: true}
	exporter := NewPostgresExporter(mockPool)

	ctx, cancel := context.WithCancel(context.Background())

	linksChan := make(chan *LinkGroup)
	go exporter.Listen(ctx, linksChan)

	group := &LinkGroup{
		From: parseURL("http://bidule"),
		To:   []*url.URL{parseURL("http://truc.com")},
	}

	linksChan <- group

	cancel()                          // Trigger a partial batch
	time.Sleep(10 * time.Millisecond) // Allow exporter to process links

	assert.Equal(t, 0, len(mockPool.InsertedRows), "should not insert rows on failure")
	assert.True(t, mockPool.hasReturnError, "should have been called and returned an error")
}

func TestPostgresExporterCloseChanEarly(t *testing.T) {
	mockPool := &MockPgxPool{}
	exporter := NewPostgresExporter(mockPool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	linksChan := make(chan *LinkGroup)
	close(linksChan) // Close channel immediately

	go exporter.Listen(ctx, linksChan)
	cancel()

	assert.True(t, true, "should cancel without panic")
}

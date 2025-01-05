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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// MockBatchResults mocks pgx.BatchResults for testing purposes.
type MockBatchResults struct {
	mu       sync.Mutex
	FailExec bool
}

func (m *MockBatchResults) Exec() (pgconn.CommandTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FailExec {
		return pgconn.NewCommandTag(""), errors.New("mocked exec failure")
	}

	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *MockBatchResults) Query() (pgx.Rows, error) {
	return nil, errors.New("Query not implemented in mock")
}

func (m *MockBatchResults) QueryRow() pgx.Row {
	return nil
}

func (m *MockBatchResults) Close() error { return nil }

// MockPgxPool mocks the MinimalPostgres interface for testing purposes.
type MockPgxPool struct {
	mu             sync.Mutex
	failSendBatch  bool
	insertedRows   [][]any
	hasReturnError bool
}

func (m *MockPgxPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failSendBatch {
		m.hasReturnError = true
		return &MockBatchResults{FailExec: true}
	}

	for _, query := range b.QueuedQueries {
		m.insertedRows = append(m.insertedRows, query.Arguments)
	}

	return &MockBatchResults{}
}

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

	assert.Equal(t, 4, len(mockPool.insertedRows), "should insert all links")
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

	assert.Equal(t, PG_BATCH_SIZE, len(mockPool.insertedRows), "should insert only a batch of PG_BATCH_SIZE")
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
	mockPool := &MockPgxPool{failSendBatch: true}
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

	assert.Equal(t, 0, len(mockPool.insertedRows), "should not insert rows on failure")
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

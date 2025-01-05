package exporter

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper to parse URL and avoid redundancy in test cases
func parseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err) // Test initialization failure should panic
	}
	return u
}

type nopeCloser struct {
	*bytes.Buffer
}

func (nopeCloser) Close() error { return nil }

func TestCSVExporterBasicFunctionality(t *testing.T) {
	buffer := &bytes.Buffer{}
	stream := nopeCloser{buffer}
	exporter := NewCSVExporter(stream)

	ctx, cancel := context.WithCancel(context.Background())

	linksChan := make(chan *LinkGroup)

	group1 := &LinkGroup{
		From: parseURL("http://example.com"),
		To:   []*url.URL{parseURL("http://example.org"), parseURL("http://example.com/login")},
	}
	group2 := &LinkGroup{
		From: parseURL("http://example.org"),
		To:   []*url.URL{parseURL("http://example.com"), parseURL("http://example.com/login")},
	}

	// Start listening in a separate goroutine
	go exporter.Listen(ctx, linksChan)

	// Send URLs to the channel
	linksChan <- group1
	linksChan <- group2

	time.Sleep(5 * time.Millisecond) // Allow some time for the exporter to process the URLs
	cancel()                         // Signal the listener to stop. This should trigger a flush
	time.Sleep(5 * time.Millisecond) // Wait for the flush and exit to happen

	// Verify written data
	csvReader := csv.NewReader(buffer)
	records, err := csvReader.ReadAll()
	assert.NoError(t, err, "should read exported CSV without error")
	assert.Len(t, records, 4, "should flush all link pair after cancelation")
}

func TestCSVExporterFlush(t *testing.T) {
	buffer := &bytes.Buffer{}
	stream := nopeCloser{buffer}
	exporter := NewCSVExporter(stream)
	size := CSV_BATCH_SIZE + 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	linksChan := make(chan *LinkGroup)

	// Start listening in a separate goroutine
	go exporter.Listen(ctx, linksChan)

	// Send enough url to trigger a flush
	for i := 0; i < size; i++ {
		linksChan <- &LinkGroup{
			From: parseURL("http://example.com"),
			To:   []*url.URL{parseURL("http://example.org")},
		}
	}

	// Allow some time for the exporter to process the URLs
	time.Sleep(10 * time.Millisecond)

	// Verify the buffer is flushed
	csvReader := csv.NewReader(buffer)
	records, err := csvReader.ReadAll()
	assert.NoError(t, err, "should read exported CSV without error")
	assert.Len(t, records, CSV_BATCH_SIZE, "should export all records up to CSV_BATCH_SIZE")

}

func TestCSVExporterContextCancellation(t *testing.T) {
	buffer := &bytes.Buffer{}
	stream := nopeCloser{buffer}
	exporter := NewCSVExporter(stream)

	ctx, cancel := context.WithCancel(context.Background())

	linksChan := make(chan *LinkGroup)
	defer close(linksChan)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		fmt.Println("listening")
		exporter.Listen(ctx, linksChan)
		fmt.Println("dooooone")
	}()

	cancel() // Cancel the context

	// Verify that the goroutine exit
	timeout := time.After(100 * time.Millisecond)
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-timeout:
		fmt.Println("heeeer")
		assert.Fail(t, "goroutine did not exit in time after cancelation")
	case <-done:
		fmt.Print("finale DOne\n")
		return
	}
}

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

	urlChan := make(chan url.URL, 2)

	url1 := *parseURL("http://example.com")
	url2 := *parseURL("http://example.org")

	// Start listening in a separate goroutine
	go exporter.Listen(ctx, urlChan)

	// Send URLs to the channel
	urlChan <- url1
	urlChan <- url2

	time.Sleep(5 * time.Millisecond) // Allow some time for the exporter to process the URLs
	cancel()                         // Signal the listener to stop. This should trigger a flush
	time.Sleep(5 * time.Millisecond) // Wait for the flush and exit to happen

	// Verify written data
	csvReader := csv.NewReader(buffer)
	records, err := csvReader.ReadAll()
	assert.NoError(t, err, "should read exported CSV without error")
	assert.Equal(t, [][]string{
		{url1.String()},
		{url2.String()},
	}, records, "exported records should match input URLs")
}

func TestCSVExporterFlush(t *testing.T) {
	buffer := &bytes.Buffer{}
	stream := nopeCloser{buffer}
	exporter := NewCSVExporter(stream)
	size := CSV_BATCH_SIZE + 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlChan := make(chan url.URL)

	// Start listening in a separate goroutine
	go exporter.Listen(ctx, urlChan)

	// Send enough url to trigger a flush
	for i := 0; i < size; i++ {
		urlChan <- *parseURL("http://example.com")
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

	urlChan := make(chan url.URL)
	defer close(urlChan)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		fmt.Println("listening")
		exporter.Listen(ctx, urlChan)
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

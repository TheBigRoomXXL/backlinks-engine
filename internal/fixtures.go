package internal

import (
	"context"
	"net/http"
)

type MockTransport struct {
	Response *http.Response
	Err      error
	NbCall   int
	callback func(*MockTransport)
}

func NewMockTransport(resp *http.Response, err error) *MockTransport {
	return &MockTransport{resp, err, 0, nil}
}

func NewMockTransportWithCallback(resp *http.Response, err error, callback func(*MockTransport)) *MockTransport {
	return &MockTransport{resp, err, 0, callback}
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.callback != nil {
		m.callback(m)
	}
	m.NbCall++
	return m.Response, m.Err
}

// Implement the fetcher interface to mock during tests
type TestFetcher struct {
	c *http.Client
}

// The fetcher will return the given response and error when Get or HEad is called
func NewTestFetcher(resp *http.Response, err error) *TestFetcher {
	return &TestFetcher{c: &http.Client{Transport: NewMockTransport(resp, err)}}
}

func (f *TestFetcher) Get(ctx context.Context, url string) (*http.Response, error) {
	return f.c.Get(url)
}

func (f *TestFetcher) Head(ctx context.Context, url string) (*http.Response, error) {
	return f.c.Head(url)
}

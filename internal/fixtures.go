package internal

import "net/http"

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

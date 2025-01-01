package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

type mockTransport struct {
	Response *http.Response
	Err      error
	NbCall   int
	callback func(*mockTransport)
}

func NewMockTransport(resp *http.Response, err error) *mockTransport {
	return &mockTransport{resp, err, 0, nil}
}

func NewMockTransportWithCallback(resp *http.Response, err error, callback func(*mockTransport)) *mockTransport {
	return &mockTransport{resp, err, 0, callback}
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.callback != nil {
		m.callback(m)
	}
	m.NbCall++
	return m.Response, m.Err
}

func NewResponse(statusCode int) *http.Response {
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(statusCode)
	recorder.Header().Add("Content-Type", "text/html")
	recorder.WriteString(`
	<!DOCTYPE html>
	<html lang="en">
		<head>
		<meta charset="utf-8">
		<title>Tested!</title>
		<link rel="stylesheet" href="style.css">
		</head>
		<body>
		blablabla
		</body>
	</html>
	`)
	return recorder.Result()
}

var noRetryStatus = []int{200, 201, 202, 203, 204, 205, 206, 207, 208, 400, 401, 402, 403, 404, 405, 406, 407, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418, 421, 422, 423, 501, 505, 506, 507, 508, 510, 511}

func TestCrawlClientGetSuccessfull(t *testing.T) {
	for _, statusCode := range append(retryStatus, noRetryStatus...) {
		t.Run("GET "+strconv.Itoa(statusCode), func(t *testing.T) {
			t.Parallel()
			response := NewResponse(statusCode)
			client := NewCrawlClient(NewMockTransport(response, nil), rate.Limit(100000), 0)

			result, err := client.Get("http://test.com/truc")
			if err != nil {
				t.Fatalf("Unexpected error calling CrawlClient.Get : %s", err)
			}
			if result != response {
				t.Fatalf(
					"CrawlClient.Head return a bad response : expected %s, got %s",
					response.Status,
					result.Status,
				)
			}
		})
	}
}

func TestCrawlClientHeadSuccessfull(t *testing.T) {
	for _, statusCode := range append(retryStatus, noRetryStatus...) {
		t.Run("HEAD "+strconv.Itoa(statusCode), func(t *testing.T) {
			t.Parallel()

			// Setup
			response := NewResponse(statusCode)
			client := NewCrawlClient(NewMockTransport(response, nil), rate.Limit(100000), 0)

			// Test
			result, err := client.Head("http://test.com/truc")
			if err != nil {
				t.Fatalf("Unexpected error calling CrawlClient.Head : %s", err)
			}
			if result != response {
				t.Fatalf(
					"CrawlClient.Head return a bad response : expected %s, got %s",
					response.Status,
					result.Status,
				)
			}
		})
	}
}

func TestCrawlClientGetFailed(t *testing.T) {
	// Setup
	err := errors.New("test-error")
	client := NewCrawlClient(NewMockTransport(nil, err), rate.Limit(100000), 0)

	// Test
	result, errResult := client.Get("http://test.com/truc")
	if !strings.Contains(errResult.Error(), err.Error()) {
		t.Fatalf("Bad error from CrawlClient : want %s ; got %s", err, errResult)
	}
	if result != nil {
		t.Fatalf("Unexepted response from CrawlClient, want nil; got %s", result.Status)
	}
}

func TestCrawlClientHeadFailed(t *testing.T) {
	// Setup
	err := errors.New("test-error")
	client := NewCrawlClient(NewMockTransport(nil, err), rate.Limit(100000), 0)

	// Test
	result, errResult := client.Head("http://test.com/truc")
	if !strings.Contains(errResult.Error(), err.Error()) {
		t.Fatalf("Bad error from CrawlClient : want %s ; got %s", err, errResult)
	}
	if result != nil {
		t.Fatalf("Unexepted response from CrawlClient, want nil; got %s", result.Status)
	}
}

func TestCrawlRateLimitGet(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Get("http://test.com/truc")
	client.Get("http://test.com/truc")
	client.Get("http://test.com/truc")
	result := time.Since(t0)
	if result < 20*time.Millisecond || result > 25*time.Millisecond {
		t.Fatalf("GET Request are not properly rate limited: 3 request in %s", result)
	}
}

func TestCrawlRateLimitGetAndHead(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Get("http://test.com/truc")
	client.Head("http://test.com/truc")
	client.Get("http://test.com/truc")
	result := time.Since(t0)
	if result < 20*time.Millisecond || result > 25*time.Millisecond {
		t.Fatalf("GET and HEAD requests do not seem to share rate limit: 3 request in %s", result)
	}
}

func TestCrawlRateLimitMultiDomainGet(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Get("http://test.com/truc")
	client.Get("http://no-test.com/truc")
	client.Get("http://another-test.com/truc")
	result := time.Since(t0)
	if result > 5*time.Millisecond {
		t.Fatalf("GET Request share rate limit between different domain: 3 request in %s", result)
	}
}

func TestCrawlRateLimitMultiDomainGet2(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Get("http://test.com/truc")
	client.Get("http://no-test.com/truc")
	client.Get("http://another-test.com/truc")
	client.Get("http://test.com/truc")
	client.Get("http://no-test.com/truc")
	client.Get("http://another-test.com/truc")
	client.Get("http://test.com/truc")
	client.Get("http://no-test.com/truc")
	client.Get("http://another-test.com/truc")
	result := time.Since(t0)
	if result < 20*time.Millisecond {
		t.Fatalf("GET Request are not properly rate limited: 9 request between 3 domains in %s", result)
	}
}

func TestCrawlRateLimitMultiDomainHead(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Head("http://test.com/truc")
	client.Head("http://no-test.com/truc")
	client.Head("http://another-test.com/truc")
	result := time.Since(t0)
	if result > 5*time.Millisecond {
		t.Fatalf("GET Request share rate limit between different domain: 3 request in %s", result)
	}
}

func TestCrawlRateLimitMultiDomainHead2(t *testing.T) {
	// Setup
	response := NewResponse(200)
	client := NewCrawlClient(
		NewMockTransport(response, nil), rate.Limit(rate.Every(10*time.Millisecond)), 0,
	)

	// Test
	t0 := time.Now()
	client.Head("http://test.com/truc")
	client.Head("http://no-test.com/truc")
	client.Head("http://another-test.com/truc")
	client.Head("http://test.com/truc")
	client.Head("http://no-test.com/truc")
	client.Head("http://another-test.com/truc")
	client.Head("http://test.com/truc")
	client.Head("http://no-test.com/truc")
	client.Head("http://another-test.com/truc")
	result := time.Since(t0)
	if result < 20*time.Millisecond {
		t.Fatalf("GET Request are not properly rate limited: 9 request between 3 domains in %s", result)
	}
}

func TestCrawlDynamicRateLimiting(t *testing.T) {
	requestFrequence := rate.Limit(10)
	mock := NewMockTransport(NewResponse(429), nil)
	client := NewCrawlClient(mock, requestFrequence, 0)

	client.Get("http://test.com/truc")
	result := client.ratelimiters["test.com"].Limit()
	if result != 5.0 {
		t.Fatalf("bad rate limit: want 50req/s; got %.2freq/s ", result)
	}
}

func TestCrawlRetryStopAfterGoodResponse(t *testing.T) {
	responseA := NewResponse(408)
	responseB := NewResponse(200)
	callback := func(m *mockTransport) {
		if m.NbCall > 0 {
			m.Response = responseB
		}
	}
	mock := NewMockTransportWithCallback(responseA, nil, callback)
	client := NewCrawlClient(mock, rate.Limit(1000000000), 10)

	responseC, err := client.Get("http://test.com/truc")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if responseC != responseB {
		t.Fatalf("bad response: want %s, got %s", responseB.Status, responseC.Status)
	}
	if mock.NbCall != 2 {
		t.Fatalf("bad number of retry: want 2, got %d", mock.NbCall)
	}
}

func TestCrawlRetryCount(t *testing.T) {
	n := 5
	for _, statusCode := range retryStatus {
		t.Run("Retry after  "+strconv.Itoa(statusCode), func(t *testing.T) {
			t.Parallel()
			response := NewResponse(statusCode)
			mock := NewMockTransport(response, nil)
			client := NewCrawlClient(mock, rate.Limit(1000000000), n)

			client.Get("http://test.com/truc")
			// +1 is for inital call
			if mock.NbCall != 5+1 {
				t.Fatalf("Retried wrong number of time: want %d, got %d", n, mock.NbCall)
			}
		})
	}
}
func TestCrawlRetryBackoff(t *testing.T) {
	response := NewResponse(408)
	requestFrequence := rate.Limit(rate.Every(10 * time.Millisecond))
	client := NewCrawlClient(NewMockTransport(response, nil), requestFrequence, 3)

	t0 := time.Now()
	client.Get("http://test.com/truc")
	result := time.Since(t0)

	// First call should be instantaneous, then wait 10ms, 100ms, 1000ms
	if result < 1110*time.Millisecond || result < 130*time.Millisecond {
		t.Fatalf("GET request do not follow exponential backoff when retrying: want ~1110ms; got %s", result)
	}
}

func TestExponentialBackoff(t *testing.T) {
	tests := map[string]struct {
		limit  rate.Limit
		retry  int
		expect float64
	}{
		"1reqPerSec/firstRetry": {
			limit:  rate.Limit(rate.Every(time.Second)),
			retry:  0,
			expect: 1,
		},
		"1reqPerSec/secondRetry": {
			limit:  rate.Limit(rate.Every(time.Second)),
			retry:  1,
			expect: 10,
		},
		"1reqPerSec/thirdRetry": {
			limit:  rate.Limit(rate.Every(time.Second)),
			retry:  2,
			expect: 100,
		},
		"0.1reqPerSec/firstRetry": {
			limit:  rate.Limit(rate.Every(10 * time.Second)),
			retry:  0,
			expect: 10,
		},
		"0.1reqPerSec/secondRetry": {
			limit:  rate.Limit(rate.Every(10 * time.Second)),
			retry:  1,
			expect: 100,
		},
		"0.1reqPerSec/thirdRetry": {
			limit:  rate.Limit(rate.Every(10 * time.Second)),
			retry:  2,
			expect: 1000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got, expected := exponentialBackoff(test.limit, test.retry), test.expect; got != expected {
				t.Fatalf("exponentialBackoff(%f, %d) failed: want %f; got %f", test.limit, test.retry, expected, got)
			}
		})
	}
}

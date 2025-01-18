package robot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/TheBigRoomXXL/backlinks-engine/internal"
)

func TestRobotGetPolicySuccess(t *testing.T) {
	// Setup
	policy := `
		User-agent: *
		Disallow: /nope*
	`
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-type", "text/plain")
	recorder.WriteHeader(200)
	recorder.WriteString(policy)
	response := recorder.Result()

	mock := internal.NewMockTransport(response, nil)
	client := &http.Client{Transport: mock}
	robot := NewInMemoryRobotPolicy(client)

	// Test
	result := robot.getRobotPolicy("test.com")
	if result != policy {
		t.Fatalf("failed to get robot.txt: want '%s'; got'%s'\n (length %d vs %d)", policy, result, len(policy), len(result))
	}
}

func TestRobotGetPolicyBadResponseStatus(t *testing.T) {
	badStatus := []int{400, 401, 402, 403, 404, 405, 406, 407, 408, 425, 429, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418, 421, 422, 423, 501, 505, 506, 507, 508, 510, 511, 500, 502, 503, 504}
	// Setup
	for _, statusCode := range badStatus {
		t.Run("Robot handle "+strconv.Itoa(statusCode), func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-type", "text/plain")
			recorder.WriteHeader(500)
			response := recorder.Result()

			mock := internal.NewMockTransport(response, nil)
			client := &http.Client{Transport: mock}
			robot := NewInMemoryRobotPolicy(client)

			// Test
			result := robot.getRobotPolicy("test.com")
			if result != norobot {
				t.Fatalf("failed to handle bad status: want '%s'; got '%s'\n", norobot, result)
			}
		})
	}
}

func TestRobotGetPolicyBadContentType(t *testing.T) {
	badContentType := []string{"text/html", "application/json"}
	// Setup
	for _, ct := range badContentType {
		t.Run("Robot handle "+ct, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-type", ct)
			recorder.WriteHeader(500)
			response := recorder.Result()

			mock := internal.NewMockTransport(response, nil)
			client := &http.Client{Transport: mock}
			robot := NewInMemoryRobotPolicy(client)

			// Test
			result := robot.getRobotPolicy("test.com")
			if result != norobot {
				t.Fatalf("failed to handle bad status: want '%s'; got '%s'\n", norobot, result)
			}
		})
	}
}

func TestRobotIsAllowed(t *testing.T) {
	tests := map[string]struct {
		path      string
		robotTxt  string
		IsAllowed bool
	}{
		"no robot.txt policy": {
			path:      "/",
			robotTxt:  "",
			IsAllowed: true,
		},
		"dIsAllowed all": {
			path:      "/",
			robotTxt:  "User-agent: *\nDisallow: /",
			IsAllowed: false,
		},
		"dIsAllowed backlink bot": {
			path:      "/",
			robotTxt:  "User-agent: BacklinksBot\nDisallow: /",
			IsAllowed: false,
		},
		"dIsAllowed sub-directory": {
			path:      "/foo/thing.html",
			robotTxt:  "User-agent: *\nDisallow: /foo/",
			IsAllowed: false,
		},
		"dIsAllowed page": {
			path:      "/thing.html",
			robotTxt:  "User-agent: *\nDisallow: /thing.html",
			IsAllowed: false,
		},
		"dIsAllowed wildcard": {
			path:      "/thing.html",
			robotTxt:  "User-agent: *\nDisallow: /thing*",
			IsAllowed: false,
		},
		"allowed sub-directory": {
			path:      "/bar/thing.html",
			robotTxt:  "User-agent: *\nDisallow: /foo/",
			IsAllowed: true,
		},
		"allowed wildcard": {
			path:      "/not-thing.html",
			robotTxt:  "User-agent: *\nDisallow: /thing.*",
			IsAllowed: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Setup
			robot := NewInMemoryRobotPolicy(http.DefaultClient)
			robot.robotPolicies.Store("test.com", test.robotTxt)

			url := &url.URL{Scheme: "http", Host: "test.com", Path: test.path}
			result := robot.IsAllowed(context.Background(), url)
			if result != test.IsAllowed {
				t.Fatalf("robot.txt rule not respected: want %t, go %t", test.IsAllowed, result)
			}
		})

	}
}

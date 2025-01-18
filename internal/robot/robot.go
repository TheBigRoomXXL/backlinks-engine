package robot

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/jimsmart/grobotstxt"
)

const norobot = "#failed-to-get-robot.txt"

type RobotPolicy interface {
	IsAllowed(*url.URL) bool
}

type InMemoryRobotPolicy struct {
	client        client.Fetcher
	locks         *sync.Map
	robotPolicies *sync.Map
}

func NewInMemoryRobotPolicy(fetcher client.Fetcher) *InMemoryRobotPolicy {
	return &InMemoryRobotPolicy{
		client:        fetcher,
		locks:         &sync.Map{},
		robotPolicies: &sync.Map{},
	}
}

func (r *InMemoryRobotPolicy) IsAllowed(ctx context.Context, url *url.URL) bool {
	_, span := telemetry.Tracer.Start(ctx, "IsAllowed")
	defer span.End()

	// This double locking is kind of terrible but I could not find a better way to escure
	// strictly one execution of getRobotPolicy (to use LoadOrStore you must have the value
	// before hand but what i want is actually to fetch the value only if needed)
	hostname := url.Hostname()
	anymu, _ := r.locks.LoadOrStore(hostname, &sync.Mutex{})
	mu := anymu.(*sync.Mutex)
	mu.Lock()
	robotTxt, ok := r.robotPolicies.Load(hostname)
	if !ok {
		robotTxt = r.getRobotPolicy(hostname)
		r.robotPolicies.Store(hostname, robotTxt)
	}
	mu.Unlock()

	robotTxtStr := robotTxt.(string)
	return grobotstxt.AgentAllowed(robotTxtStr, "BacklinksBot", url.String())
}

func (r *InMemoryRobotPolicy) getRobotPolicy(hostname string) string {
	resp, err := r.client.Get("http://" + hostname + "/robots.txt")
	if err != nil {
		slog.Warn(fmt.Sprintf("failed to get robot.txt for %s: %s", hostname, err))
		return norobot
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn(fmt.Sprintf("failed to get robot.txt for %s: response with status %d", hostname, resp.StatusCode))
		return norobot
	}

	contentType := strings.ToLower(resp.Header.Get("content-Type"))
	if !strings.Contains(contentType, "text/plain") {
		slog.Warn(fmt.Sprintf("failed to get robot.txt for %s: response with content-type %s", hostname, contentType))
		return norobot
	}

	data := make([]byte, 512*1024)
	n, err := resp.Body.Read(data)
	if err != nil && err != io.EOF {
		slog.Warn(fmt.Sprintf("failed to get robot.txt for %s: failed to read body: %s", hostname, err))
		return norobot
	}
	return string(data[:n])
}

package commons

import (
	"context"
	"slices"
	"strings"
	"time"
)

func ReverseHostname(hostname string) string {
	labels := strings.Split(hostname, ".")
	slices.Reverse(labels)
	return strings.Join(labels, ".")
}

// Delay returns nil after the specified duration or error if interrupted.
func Delay(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	select {
	case <-ctx.Done():
		t.Stop()
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

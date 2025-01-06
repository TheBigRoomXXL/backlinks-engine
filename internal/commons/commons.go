package commons

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

type LinkGroup struct {
	From *url.URL
	To   []*url.URL
}
type Link struct {
	From *url.URL
	To   *url.URL
}

func ReverseHostname(hostname string) string {
	labels := strings.Split(hostname, ".")
	slices.Reverse(labels)
	return strings.Join(labels, ".")
}

func NormalizeUrl(url *url.URL) (*url.URL, error) {
	// TODO: bring back the rules from GoogleSafeBrowsing in a performant way
	// url, err := canonicalizer.GoogleSafeBrowsing.Parse(urlRaw)
	if url.Scheme == "" {
		url.Scheme = "http"
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return nil, fmt.Errorf("url scheme is not http or https: %s", url.Scheme)
	}

	p := url.Port()
	if p != "" && p != "80" && p != "443" {
		return nil, fmt.Errorf("port is not 80 or 443: %s", p)
	}
	url.Host = url.Hostname()
	url.Fragment = ""
	url.RawQuery = ""

	return url, nil
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

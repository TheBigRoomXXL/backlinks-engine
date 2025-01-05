package exporter

import (
	"context"
	"net/url"
)

type LinkGroup struct {
	From *url.URL
	To   []*url.URL
}

// Exporter receive the extracted and normalized url and process them asynchronously.
type Exporter interface {
	Listen(context.Context, chan *LinkGroup)
}

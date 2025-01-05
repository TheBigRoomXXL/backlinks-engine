package exporter

import (
	"context"
	"net/url"
)

// Exporter receive the extracted and normalized url and process them asynchronously.
type Exporter interface {
	Listen(context.Context, chan url.URL)
}

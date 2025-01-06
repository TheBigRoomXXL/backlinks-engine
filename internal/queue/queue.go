package queue

import (
	"net/url"
)

type Queue interface {
	Add(*url.URL) error
	Next() (*url.URL, error)
}

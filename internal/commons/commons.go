package commons

import (
	"slices"
	"strings"
)

func ReverseHostname(hostname string) string {
	labels := strings.Split(hostname, ".")
	slices.Reverse(labels)
	return strings.Join(labels, ".")
}

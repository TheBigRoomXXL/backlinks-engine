package commons

import (
	"net/url"
	"testing"
)

func TestReverseHostname(t *testing.T) {
	tests := map[string]struct {
		input  string
		result string
	}{
		"one label": {
			input:  "test.com",
			result: "com.test",
		},
		"two label": {
			input:  "truc.test.com",
			result: "com.test.truc",
		},
		"already reversed, one label": {
			input:  "com.test",
			result: "test.com",
		},
		"already reversed, two label": {
			input:  "com.test.truc",
			result: "truc.test.com",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got, expected := ReverseHostname(test.input), test.result; got != expected {
				t.Fatalf("ReverseHostname(%q) returned %q; expected %q", test.input, got, expected)
			}
		})
	}
}

func TestNornalizeURL(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			input:  "test.com",
			output: "http://test.com",
		},
		{
			input:  "http://test.com:80",
			output: "http://test.com",
		},
		{
			input:  "https://test.com:443",
			output: "https://test.com",
		},
		{
			input:  "http://test.com/truc#machin",
			output: "http://test.com/truc",
		},
		{
			input:  "http://test.com/truc?machin=bidule",
			output: "http://test.com/truc",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			url, err := url.Parse(test.input)
			if err != nil {
				t.Fatal("invalid test input")
			}
			got, err := NormalizeUrl(url)
			if err != nil {
				t.Fatalf("unexpected error during test: %s", err)
			}
			if got.String() != test.output {
				t.Fatalf("NormalizeUrl(%s) failed: want %s; got %s", test.input, test.output, got)
			}
		})
	}
}

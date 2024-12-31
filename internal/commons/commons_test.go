package commons

import "testing"

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

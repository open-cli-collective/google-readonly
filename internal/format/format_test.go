package format

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"long string truncated", "hello world", 8, "hello..."},
		{"empty string", "", 10, ""},
		{"minimum truncation", "abcdefgh", 4, "a..."},
		{"handles small maxLen", "hello", 2, "h..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Truncate(tt.input, tt.maxLen)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes with decimal", 2621440, "2.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Size(tt.bytes)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

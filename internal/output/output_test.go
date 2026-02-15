package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected string
	}{
		{
			name:     "simple struct",
			data:     struct{ Name string }{"test"},
			expected: "{\n  \"Name\": \"test\"\n}\n",
		},
		{
			name:     "slice",
			data:     []int{1, 2, 3},
			expected: "[\n  1,\n  2,\n  3\n]\n",
		},
		{
			name:     "map",
			data:     map[string]int{"a": 1},
			expected: "{\n  \"a\": 1\n}\n",
		},
		{
			name:     "nil",
			data:     nil,
			expected: "null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := JSON(&buf, tt.data)
			testutil.NoError(t, err)
			testutil.Equal(t, buf.String(), tt.expected)
		})
	}
}

func TestJSON_indentation(t *testing.T) {
	data := struct {
		Nested struct {
			Value string
		}
	}{
		Nested: struct{ Value string }{Value: "deep"},
	}

	var buf bytes.Buffer
	err := JSON(&buf, data)
	testutil.NoError(t, err)

	// Check that indentation uses 2 spaces
	lines := strings.Split(buf.String(), "\n")
	testutil.True(t, strings.HasPrefix(lines[1], "  "))
	testutil.True(t, strings.HasPrefix(lines[2], "    "))
}

func TestJSON_error(t *testing.T) {
	// Channels cannot be encoded to JSON
	data := make(chan int)
	var buf bytes.Buffer

	err := JSON(&buf, data)
	testutil.Error(t, err)
}

package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
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
	require.NoError(t, err)

	// Check that indentation uses 2 spaces
	lines := strings.Split(buf.String(), "\n")
	assert.True(t, strings.HasPrefix(lines[1], "  "), "expected 2-space indentation")
	assert.True(t, strings.HasPrefix(lines[2], "    "), "expected 4-space indentation for nested")
}

func TestJSON_error(t *testing.T) {
	// Channels cannot be encoded to JSON
	data := make(chan int)
	var buf bytes.Buffer

	err := JSON(&buf, data)
	assert.Error(t, err)
}

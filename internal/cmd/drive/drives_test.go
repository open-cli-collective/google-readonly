package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDrivesCommand(t *testing.T) {
	cmd := newDrivesCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "drives", cmd.Use)
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has refresh flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("refresh")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "shared drives")
	})

	t.Run("has long description", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Shared Drives")
		assert.Contains(t, cmd.Long, "cache")
	})
}

func TestLooksLikeDriveID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid shared drive ID",
			input:    "0ALengineering123456",
			expected: true,
		},
		{
			name:     "another valid ID",
			input:    "0ALsomeDriveIdHere789",
			expected: true,
		},
		{
			name:     "short string is not ID",
			input:    "0AL",
			expected: false,
		},
		{
			name:     "regular name is not ID",
			input:    "Engineering",
			expected: false,
		},
		{
			name:     "name with spaces",
			input:    "Finance Team",
			expected: false,
		},
		{
			name:     "starts with 0A but too short",
			input:    "0Ashort",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "doesn't start with 0A",
			input:    "1ALengineering123456",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeDriveID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

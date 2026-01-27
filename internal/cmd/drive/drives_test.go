package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
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

func TestResolveDriveScope(t *testing.T) {
	t.Run("returns MyDriveOnly when myDrive flag is true", func(t *testing.T) {
		mock := &testutil.MockDriveClient{}

		scope, err := resolveDriveScope(mock, true, "")

		assert.NoError(t, err)
		assert.True(t, scope.MyDriveOnly)
		assert.False(t, scope.AllDrives)
		assert.Empty(t, scope.DriveID)
	})

	t.Run("returns AllDrives when no flags provided", func(t *testing.T) {
		mock := &testutil.MockDriveClient{}

		scope, err := resolveDriveScope(mock, false, "")

		assert.NoError(t, err)
		assert.True(t, scope.AllDrives)
		assert.False(t, scope.MyDriveOnly)
		assert.Empty(t, scope.DriveID)
	})

	t.Run("returns DriveID directly when input looks like ID", func(t *testing.T) {
		mock := &testutil.MockDriveClient{}

		scope, err := resolveDriveScope(mock, false, "0ALengineering123456")

		assert.NoError(t, err)
		assert.Equal(t, "0ALengineering123456", scope.DriveID)
		assert.False(t, scope.AllDrives)
		assert.False(t, scope.MyDriveOnly)
	})

	t.Run("resolves drive name to ID via API", func(t *testing.T) {
		mock := &testutil.MockDriveClient{
			ListSharedDrivesFunc: func(pageSize int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
					{ID: "0ALfin456", Name: "Finance"},
				}, nil
			},
		}

		scope, err := resolveDriveScope(mock, false, "Engineering")

		assert.NoError(t, err)
		assert.Equal(t, "0ALeng123", scope.DriveID)
	})

	t.Run("resolves drive name case-insensitively", func(t *testing.T) {
		mock := &testutil.MockDriveClient{
			ListSharedDrivesFunc: func(pageSize int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
				}, nil
			},
		}

		scope, err := resolveDriveScope(mock, false, "ENGINEERING")

		assert.NoError(t, err)
		assert.Equal(t, "0ALeng123", scope.DriveID)
	})

	t.Run("returns error when drive name not found", func(t *testing.T) {
		mock := &testutil.MockDriveClient{
			ListSharedDrivesFunc: func(pageSize int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
				}, nil
			},
		}

		_, err := resolveDriveScope(mock, false, "NonExistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shared drive not found")
	})
}

func TestSearchCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newSearchCommand()
		cmd.SetArgs([]string{"query", "--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestListCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newListCommand()
		cmd.SetArgs([]string{"--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestTreeCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newTreeCommand()
		cmd.SetArgs([]string{"--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
	})
}

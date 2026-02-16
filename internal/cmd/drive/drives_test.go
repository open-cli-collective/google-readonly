package drive

import (
	"context"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestDrivesCommand(t *testing.T) {
	cmd := newDrivesCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "drives")
	})

	t.Run("requires no arguments", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has refresh flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("refresh")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "shared drives")
	})

	t.Run("has long description", func(t *testing.T) {
		testutil.Contains(t, cmd.Long, "Shared Drives")
		testutil.Contains(t, cmd.Long, "cache")
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
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestResolveDriveScope(t *testing.T) {
	t.Run("returns MyDriveOnly when myDrive flag is true", func(t *testing.T) {
		mock := &MockDriveClient{}

		scope, err := resolveDriveScope(context.Background(), mock, true, "")

		testutil.NoError(t, err)
		testutil.True(t, scope.MyDriveOnly)
		testutil.False(t, scope.AllDrives)
		testutil.Empty(t, scope.DriveID)
	})

	t.Run("returns AllDrives when no flags provided", func(t *testing.T) {
		mock := &MockDriveClient{}

		scope, err := resolveDriveScope(context.Background(), mock, false, "")

		testutil.NoError(t, err)
		testutil.True(t, scope.AllDrives)
		testutil.False(t, scope.MyDriveOnly)
		testutil.Empty(t, scope.DriveID)
	})

	t.Run("returns DriveID directly when input looks like ID", func(t *testing.T) {
		mock := &MockDriveClient{}

		scope, err := resolveDriveScope(context.Background(), mock, false, "0ALengineering123456")

		testutil.NoError(t, err)
		testutil.Equal(t, scope.DriveID, "0ALengineering123456")
		testutil.False(t, scope.AllDrives)
		testutil.False(t, scope.MyDriveOnly)
	})

	t.Run("resolves drive name to ID via API", func(t *testing.T) {
		mock := &MockDriveClient{
			ListSharedDrivesFunc: func(_ context.Context, _ int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
					{ID: "0ALfin456", Name: "Finance"},
				}, nil
			},
		}

		scope, err := resolveDriveScope(context.Background(), mock, false, "Engineering")

		testutil.NoError(t, err)
		testutil.Equal(t, scope.DriveID, "0ALeng123")
	})

	t.Run("resolves drive name case-insensitively", func(t *testing.T) {
		mock := &MockDriveClient{
			ListSharedDrivesFunc: func(_ context.Context, _ int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
				}, nil
			},
		}

		scope, err := resolveDriveScope(context.Background(), mock, false, "ENGINEERING")

		testutil.NoError(t, err)
		testutil.Equal(t, scope.DriveID, "0ALeng123")
	})

	t.Run("returns error when drive name not found", func(t *testing.T) {
		mock := &MockDriveClient{
			ListSharedDrivesFunc: func(_ context.Context, _ int64) ([]*drive.SharedDrive, error) {
				return []*drive.SharedDrive{
					{ID: "0ALeng123", Name: "Engineering"},
				}, nil
			},
		}

		_, err := resolveDriveScope(context.Background(), mock, false, "NonExistent")

		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "shared drive not found")
	})
}

func TestSearchCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newSearchCommand()
		cmd.SetArgs([]string{"query", "--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestListCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newListCommand()
		cmd.SetArgs([]string{"--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestTreeCommand_MutualExclusivity(t *testing.T) {
	t.Run("errors when both my-drive and drive flags set", func(t *testing.T) {
		cmd := newTreeCommand()
		cmd.SetArgs([]string{"--my-drive", "--drive", "Engineering"})

		err := cmd.Execute()

		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "mutually exclusive")
	})
}

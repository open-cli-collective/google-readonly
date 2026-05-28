package drive

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestStarCommand(t *testing.T) {
	cmd := newStarCommand()

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "n")
	})

	t.Run("has stdin flag", func(t *testing.T) {
		testutil.NotNil(t, cmd.Flags().Lookup("stdin"))
	})

	t.Run("has query flag", func(t *testing.T) {
		testutil.NotNil(t, cmd.Flags().Lookup("query"))
	})
}

func TestStarCommand_Success(t *testing.T) {
	var starredIDs []string
	mock := &MockDriveClient{
		StarFileFunc: func(_ context.Context, fileID string) error {
			starredIDs = append(starredIDs, fileID)
			return nil
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"file1", "file2"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Starred 2 file(s).")
		testutil.Len(t, starredIDs, 2)
	})
}

func TestStarCommand_DryRun(t *testing.T) {
	mock := &MockDriveClient{
		StarFileFunc: func(_ context.Context, _ string) error {
			t.Fatal("StarFile should not be called in dry-run")
			return nil
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"file1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run] Would star 1 file(s).")
	})
}

func TestStarCommand_Query(t *testing.T) {
	mock := &MockDriveClient{
		SearchFileIDsFunc: func(_ context.Context, q string, _ int64) ([]string, error) {
			testutil.Equal(t, q, "name contains 'report'")
			return []string{"file1", "file2", "file3"}, nil
		},
		StarFileFunc: func(_ context.Context, _ string) error {
			return nil
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"--query", "name contains 'report'"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Starred 3 file(s).")
	})
}

func TestStarCommand_APIError(t *testing.T) {
	mock := &MockDriveClient{
		StarFileFunc: func(_ context.Context, _ string) error {
			return errors.New("permission denied")
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"file1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "starring file file1")
	})
}

func TestStarCommand_ClientCreationError(t *testing.T) {
	cmd := newStarCommand()
	cmd.SetArgs([]string{"file1"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Drive client")
	})
}

func TestUnstarCommand_Success(t *testing.T) {
	var unstarredIDs []string
	mock := &MockDriveClient{
		UnstarFileFunc: func(_ context.Context, fileID string) error {
			unstarredIDs = append(unstarredIDs, fileID)
			return nil
		},
	}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"file1"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Unstarred 1 file(s).")
		testutil.Len(t, unstarredIDs, 1)
	})
}

func TestUnstarCommand_DryRun(t *testing.T) {
	mock := &MockDriveClient{
		UnstarFileFunc: func(_ context.Context, _ string) error {
			t.Fatal("UnstarFile should not be called in dry-run")
			return nil
		},
	}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"file1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run] Would unstar 1 file(s).")
	})
}

func TestUnstarCommand_APIError(t *testing.T) {
	mock := &MockDriveClient{
		UnstarFileFunc: func(_ context.Context, _ string) error {
			return errors.New("permission denied")
		},
	}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"file1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "unstarring file file1")
	})
}

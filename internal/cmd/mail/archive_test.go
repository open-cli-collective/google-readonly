package mail

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestArchiveCommand(t *testing.T) {
	cmd := newArchiveCommand()

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})

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

func TestArchiveCommand_Success(t *testing.T) {
	var modifiedIDs []string
	var removedLabels []string

	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, ids []string, _ []string, remove []string) error {
			modifiedIDs = ids
			removedLabels = remove
			return nil
		},
	}

	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"msg1", "msg2"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Archived 2 message(s).")
		testutil.Len(t, modifiedIDs, 2)
		testutil.SliceContains(t, removedLabels, "INBOX")
	})
}

func TestArchiveCommand_DryRun(t *testing.T) {
	mock := &MockGmailClient{}

	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"msg1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run] Would archive 1 message(s).")
	})
}

func TestArchiveCommand_JSON(t *testing.T) {
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, _ []string) error {
			return nil
		},
	}

	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"msg1", "--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var result bulk.Result
		err := json.Unmarshal([]byte(output), &result)
		testutil.NoError(t, err)
		testutil.Equal(t, result.Action, "archived")
		testutil.Equal(t, result.Count, 1)
	})
}

func TestArchiveCommand_Query(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessageIDsFunc: func(_ context.Context, q string, _ int64) ([]string, error) {
			testutil.Equal(t, q, "from:noreply")
			return []string{"msg1", "msg2", "msg3"}, nil
		},
		ModifyMessagesFunc: func(_ context.Context, ids []string, _, _ []string) error {
			testutil.Len(t, ids, 3)
			return nil
		},
	}

	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"--query", "from:noreply"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Archived 3 message(s).")
	})
}

func TestArchiveCommand_APIError(t *testing.T) {
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, _ []string) error {
			return errors.New("permission denied")
		},
	}

	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"msg1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "archiving messages")
	})
}

func TestArchiveCommand_ClientCreationError(t *testing.T) {
	cmd := newArchiveCommand()
	cmd.SetArgs([]string{"msg1"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Gmail client")
	})
}

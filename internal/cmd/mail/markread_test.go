package mail

import (
	"context"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestMarkReadCommand(t *testing.T) {
	cmd := newMarkReadCommand()

	t.Run("has dry-run flag", func(t *testing.T) {
		testutil.NotNil(t, cmd.Flags().Lookup("dry-run"))
	})
}

func TestMarkReadCommand_Success(t *testing.T) {
	var removedLabels []string
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, remove []string) error {
			removedLabels = remove
			return nil
		},
	}

	cmd := newMarkReadCommand()
	cmd.SetArgs([]string{"msg1", "msg2"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Marked as read 2 message(s).")
		testutil.SliceContains(t, removedLabels, "UNREAD")
	})
}

func TestMarkReadCommand_DryRun(t *testing.T) {
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, _ []string) error {
			t.Fatal("ModifyMessages should not be called in dry-run")
			return nil
		},
	}
	cmd := newMarkReadCommand()
	cmd.SetArgs([]string{"msg1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would mark as read 1 message(s).")
	})
}

func TestMarkUnreadCommand_Success(t *testing.T) {
	var addedLabels []string
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, add, _ []string) error {
			addedLabels = add
			return nil
		},
	}

	cmd := newMarkUnreadCommand()
	cmd.SetArgs([]string{"msg1"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Marked as unread 1 message(s).")
		testutil.SliceContains(t, addedLabels, "UNREAD")
	})
}

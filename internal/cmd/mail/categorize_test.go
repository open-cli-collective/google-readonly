package mail

import (
	"context"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestCategorizeCommand(t *testing.T) {
	cmd := newCategorizeCommand()

	t.Run("requires at least one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)
	})

}

func TestCategorizeCommand_Success(t *testing.T) {
	var addedLabels, removedLabels []string
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, add, remove []string) error {
			addedLabels = add
			removedLabels = remove
			return nil
		},
	}

	cmd := newCategorizeCommand()
	cmd.SetArgs([]string{"promotions", "msg1"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Recategorized to promotions 1 message(s).")
		testutil.SliceContains(t, addedLabels, "CATEGORY_PROMOTIONS")
		// Should remove all category labels
		testutil.Greater(t, len(removedLabels), 0)
	})
}

func TestCategorizeCommand_InvalidCategory(t *testing.T) {
	mock := &MockGmailClient{}

	cmd := newCategorizeCommand()
	cmd.SetArgs([]string{"invalid", "msg1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid category")
	})
}

func TestCategorizeCommand_DryRun(t *testing.T) {
	mock := &MockGmailClient{}

	cmd := newCategorizeCommand()
	cmd.SetArgs([]string{"social", "msg1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would recategorize to social 1 message(s).")
	})
}

func TestCategorizeCommand_CaseInsensitive(t *testing.T) {
	mock := &MockGmailClient{
		ModifyMessagesFunc: func(_ context.Context, _ []string, add, _ []string) error {
			testutil.SliceContains(t, add, "CATEGORY_FORUMS")
			return nil
		},
	}

	cmd := newCategorizeCommand()
	cmd.SetArgs([]string{"Forums", "msg1"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Recategorized to forums 1 message(s).")
	})
}

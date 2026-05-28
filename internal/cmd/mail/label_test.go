package mail

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestLabelCommand(t *testing.T) {
	cmd := newLabelCommand()

	t.Run("requires at least one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)
	})

	t.Run("accepts label name only", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"Work"})
		testutil.NoError(t, err)
	})

}

func TestLabelCommand_Success(t *testing.T) {
	var addedLabels []string
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, name string) (string, error) {
			testutil.Equal(t, name, "Work")
			return "Label_1", nil
		},
		ModifyMessagesFunc: func(_ context.Context, _ []string, add, _ []string) error {
			addedLabels = add
			return nil
		},
	}

	cmd := newLabelCommand()
	cmd.SetArgs([]string{"Work", "msg1", "msg2"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Added label 'Work' to 2 message(s).")
		testutil.SliceContains(t, addedLabels, "Label_1")
	})
}

func TestLabelCommand_LabelNotFound(t *testing.T) {
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("label \"Nonexistent\" not found")
		},
	}

	cmd := newLabelCommand()
	cmd.SetArgs([]string{"Nonexistent", "msg1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "resolving label")
	})
}

func TestLabelCommand_DryRun(t *testing.T) {
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, _ string) (string, error) {
			return "Label_1", nil
		},
	}

	cmd := newLabelCommand()
	cmd.SetArgs([]string{"Work", "msg1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would add label 'Work' to 1 message(s).")
	})
}

func TestUnlabelCommand_Success(t *testing.T) {
	var removedLabels []string
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, name string) (string, error) {
			testutil.Equal(t, name, "Work")
			return "Label_1", nil
		},
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, remove []string) error {
			removedLabels = remove
			return nil
		},
	}

	cmd := newUnlabelCommand()
	cmd.SetArgs([]string{"Work", "msg1"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Removed label 'Work' from 1 message(s).")
		testutil.SliceContains(t, removedLabels, "Label_1")
	})
}

func TestUnlabelCommand_DryRun(t *testing.T) {
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, _ string) (string, error) {
			return "Label_1", nil
		},
	}

	cmd := newUnlabelCommand()
	cmd.SetArgs([]string{"Work", "msg1", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would remove label 'Work' from 1 message(s).")
	})
}

func TestUnlabelCommand_LabelNotFound(t *testing.T) {
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("label \"Nope\" not found")
		},
	}

	cmd := newUnlabelCommand()
	cmd.SetArgs([]string{"Nope", "msg1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "resolving label")
	})
}

func TestUnlabelCommand_APIError(t *testing.T) {
	mock := &MockGmailClient{
		GetLabelIDFunc: func(_ context.Context, _ string) (string, error) {
			return "Label_1", nil
		},
		ModifyMessagesFunc: func(_ context.Context, _ []string, _, _ []string) error {
			return errors.New("API error")
		},
	}

	cmd := newUnlabelCommand()
	cmd.SetArgs([]string{"Work", "msg1"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "removing label")
	})
}

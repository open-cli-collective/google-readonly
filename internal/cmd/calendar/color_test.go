package calendar

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestColorCommand(t *testing.T) {
	cmd := newColorCommand()

	t.Run("has dry-run flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("dry-run")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "n")
	})

	t.Run("has calendar flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "c")
	})
}

func TestColorCommand_ByName(t *testing.T) {
	var capturedColorID string
	mock := &MockCalendarClient{
		SetEventColorFunc: func(_ context.Context, _, _, colorID string) error {
			capturedColorID = colorID
			return nil
		},
	}

	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "tomato"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, capturedColorID, "11")
		testutil.Contains(t, output, "Set event event123 color to tomato.")
	})
}

func TestColorCommand_ByID(t *testing.T) {
	var capturedColorID string
	mock := &MockCalendarClient{
		SetEventColorFunc: func(_ context.Context, _, _, colorID string) error {
			capturedColorID = colorID
			return nil
		},
	}

	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "7"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, capturedColorID, "7")
		testutil.Contains(t, output, "Set event event123 color to peacock.")
	})
}

func TestColorCommand_InvalidColor(t *testing.T) {
	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "purple"})

	withMockClient(&MockCalendarClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid color")
	})
}

func TestColorCommand_InvalidNumericID(t *testing.T) {
	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "99"})

	withMockClient(&MockCalendarClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid color")
	})
}

func TestColorCommand_DryRun(t *testing.T) {
	mock := &MockCalendarClient{
		SetEventColorFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("SetEventColor should not be called in dry-run")
			return nil
		},
	}

	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "sage", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would set event event123 color to sage.")
	})
}

func TestColorCommand_APIError(t *testing.T) {
	mock := &MockCalendarClient{
		SetEventColorFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("permission denied")
		},
	}

	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "tomato"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "setting event color")
	})
}

func TestColorCommand_ClientCreationError(t *testing.T) {
	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "tomato"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Calendar client")
	})
}

func TestColorCommand_CaseInsensitive(t *testing.T) {
	mock := &MockCalendarClient{
		SetEventColorFunc: func(_ context.Context, _, _, colorID string) error {
			testutil.Equal(t, colorID, "11")
			return nil
		},
	}

	cmd := newColorCommand()
	cmd.SetArgs([]string{"event123", "Tomato"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "tomato")
	})
}

func TestResolveColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		wantID    string
		wantName  string
		wantError bool
	}{
		{"tomato", "11", "tomato", false},
		{"lavender", "1", "lavender", false},
		{"1", "1", "lavender", false},
		{"11", "11", "tomato", false},
		{"0", "", "", true},
		{"12", "", "", true},
		{"purple", "", "", true},
		{"abc", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			id, name, err := resolveColor(tt.input)
			if tt.wantError {
				testutil.Error(t, err)
			} else {
				testutil.NoError(t, err)
				testutil.Equal(t, id, tt.wantID)
				testutil.Equal(t, name, tt.wantName)
			}
		})
	}
}

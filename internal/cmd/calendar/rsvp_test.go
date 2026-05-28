package calendar

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestRSVPCommand(t *testing.T) {
	cmd := newRSVPCommand()

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

func TestRSVPCommand_Accept(t *testing.T) {
	var capturedCalID, capturedEventID, capturedResponse string
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, calID, eventID, response string) error {
			capturedCalID = calID
			capturedEventID = eventID
			capturedResponse = response
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "accept"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, capturedCalID, "primary")
		testutil.Equal(t, capturedEventID, "event123")
		testutil.Equal(t, capturedResponse, "accepted")
		testutil.Contains(t, output, "RSVP'd 'accepted' to event event123.")
	})
}

func TestRSVPCommand_Decline(t *testing.T) {
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, _, _, response string) error {
			testutil.Equal(t, response, "declined")
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "decline"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "RSVP'd 'declined'")
	})
}

func TestRSVPCommand_Tentative(t *testing.T) {
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, _, _, response string) error {
			testutil.Equal(t, response, "tentative")
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "tentative"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "RSVP'd 'tentative'")
	})
}

func TestRSVPCommand_InvalidResponse(t *testing.T) {
	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "maybe"})

	withMockClient(&MockCalendarClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid response")
	})
}

func TestRSVPCommand_DryRun(t *testing.T) {
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, _, _, _ string) error {
			t.Fatal("RSVPEvent should not be called in dry-run")
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "accept", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "[dry-run] Would RSVP 'accepted' to event event123.")
	})
}

func TestRSVPCommand_WithCalendar(t *testing.T) {
	var capturedCalID string
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, calID, _, _ string) error {
			capturedCalID = calID
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "accept", "--calendar", "work@group.calendar.google.com"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, capturedCalID, "work@group.calendar.google.com")
		testutil.Contains(t, output, "RSVP'd 'accepted'")
	})
}

func TestRSVPCommand_APIError(t *testing.T) {
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, _, _, _ string) error {
			return errors.New("not an attendee")
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "accept"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "updating RSVP")
	})
}

func TestRSVPCommand_ClientCreationError(t *testing.T) {
	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "accept"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Calendar client")
	})
}

func TestRSVPCommand_CaseInsensitive(t *testing.T) {
	mock := &MockCalendarClient{
		RSVPEventFunc: func(_ context.Context, _, _, response string) error {
			testutil.Equal(t, response, "accepted")
			return nil
		},
	}

	cmd := newRSVPCommand()
	cmd.SetArgs([]string{"event123", "Accept"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "RSVP'd 'accepted'")
	})
}

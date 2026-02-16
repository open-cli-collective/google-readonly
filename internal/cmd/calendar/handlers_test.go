package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"google.golang.org/api/calendar/v3"

	calendarapi "github.com/open-cli-collective/google-readonly/internal/calendar"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// withMockClient sets up a mock client factory for tests
func withMockClient(mock CalendarClient, f func()) {
	testutil.WithFactory(&ClientFactory, func(_ context.Context) (CalendarClient, error) {
		return mock, nil
	}, f)
}

// withFailingClientFactory sets up a factory that returns an error
func withFailingClientFactory(f func()) {
	testutil.WithFactory(&ClientFactory, func(_ context.Context) (CalendarClient, error) {
		return nil, errors.New("connection failed")
	}, f)
}

func TestListCommand_Success(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(_ context.Context) ([]*calendar.CalendarListEntry, error) {
			return testutil.SampleCalendars(), nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "primary@example.com")
		testutil.Contains(t, output, "(primary)")
		testutil.Contains(t, output, "work@example.com")
	})
}

func TestListCommand_JSONOutput(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(_ context.Context) ([]*calendar.CalendarListEntry, error) {
			return testutil.SampleCalendars(), nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var calendars []*calendarapi.CalendarInfo
		err := json.Unmarshal([]byte(output), &calendars)
		testutil.NoError(t, err)
		testutil.Len(t, calendars, 2)
	})
}

func TestListCommand_Empty(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(_ context.Context) ([]*calendar.CalendarListEntry, error) {
			return []*calendar.CalendarListEntry{}, nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "No calendars found")
	})
}

func TestListCommand_Empty_JSON(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(_ context.Context) ([]*calendar.CalendarListEntry, error) {
			return []*calendar.CalendarListEntry{}, nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var calendars []any
		err := json.Unmarshal([]byte(output), &calendars)
		testutil.NoError(t, err)
		testutil.Len(t, calendars, 0)
	})
}

func TestListCommand_APIError(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(_ context.Context) ([]*calendar.CalendarListEntry, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "listing calendars")
	})
}

func TestListCommand_ClientCreationError(t *testing.T) {
	cmd := newListCommand()

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Calendar client")
	})
}

func TestEventsCommand_Success(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, calendarID, _, _ string, _ int64) ([]*calendar.Event, error) {
			testutil.Equal(t, calendarID, "primary")
			return []*calendar.Event{testutil.SampleEvent("event1")}, nil
		},
	}

	cmd := newEventsCommand()
	cmd.SetArgs([]string{}) // Uses default "primary" calendar

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Test Meeting")
	})
}

func TestEventsCommand_WithDateRange(t *testing.T) {
	var capturedTimeMin, capturedTimeMax string
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, timeMin, timeMax string, _ int64) ([]*calendar.Event, error) {
			capturedTimeMin = timeMin
			capturedTimeMax = timeMax
			return []*calendar.Event{}, nil
		},
	}

	cmd := newEventsCommand()
	cmd.SetArgs([]string{"--from", "2024-01-01", "--to", "2024-01-31"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		// Verify dates were parsed and passed
		testutil.Contains(t, capturedTimeMin, "2024-01-01")
		testutil.Contains(t, capturedTimeMax, "2024-01-31")
		testutil.Contains(t, output, "No events")
	})
}

func TestEventsCommand_JSONOutput(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, _, _ string, _ int64) ([]*calendar.Event, error) {
			return []*calendar.Event{testutil.SampleEvent("event1")}, nil
		},
	}

	cmd := newEventsCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var events []*calendarapi.Event
		err := json.Unmarshal([]byte(output), &events)
		testutil.NoError(t, err)
		testutil.Len(t, events, 1)
	})
}

func TestEventsCommand_Empty_JSON(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, _, _ string, _ int64) ([]*calendar.Event, error) {
			return []*calendar.Event{}, nil
		},
	}

	cmd := newEventsCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var events []any
		err := json.Unmarshal([]byte(output), &events)
		testutil.NoError(t, err)
		testutil.Len(t, events, 0)
	})
}

func TestEventsCommand_InvalidFromDate(t *testing.T) {
	cmd := newEventsCommand()
	cmd.SetArgs([]string{"--from", "invalid-date"})

	withMockClient(&MockCalendarClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid --from date")
	})
}

func TestEventsCommand_InvalidToDate(t *testing.T) {
	cmd := newEventsCommand()
	cmd.SetArgs([]string{"--to", "invalid-date"})

	withMockClient(&MockCalendarClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "invalid --to date")
	})
}

func TestGetCommand_Success(t *testing.T) {
	mock := &MockCalendarClient{
		GetEventFunc: func(_ context.Context, calendarID, eventID string) (*calendar.Event, error) {
			testutil.Equal(t, calendarID, "primary")
			testutil.Equal(t, eventID, "event123")
			return testutil.SampleEvent("event123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"event123"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "event123")
		testutil.Contains(t, output, "Test Meeting")
		testutil.Contains(t, output, "Conference Room A")
	})
}

func TestGetCommand_JSONOutput(t *testing.T) {
	mock := &MockCalendarClient{
		GetEventFunc: func(_ context.Context, _, _ string) (*calendar.Event, error) {
			return testutil.SampleEvent("event123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"event123", "--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var event calendarapi.Event
		err := json.Unmarshal([]byte(output), &event)
		testutil.NoError(t, err)
		testutil.Equal(t, event.ID, "event123")
	})
}

func TestGetCommand_NotFound(t *testing.T) {
	mock := &MockCalendarClient{
		GetEventFunc: func(_ context.Context, _, _ string) (*calendar.Event, error) {
			return nil, errors.New("event not found")
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "getting event")
	})
}

func TestTodayCommand_Success(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, _, _ string, _ int64) ([]*calendar.Event, error) {
			return []*calendar.Event{testutil.SampleEvent("today_event")}, nil
		},
	}

	cmd := newTodayCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Test Meeting")
	})
}

func TestTodayCommand_Empty_JSON(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, _, _ string, _ int64) ([]*calendar.Event, error) {
			return []*calendar.Event{}, nil
		},
	}

	cmd := newTodayCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var events []any
		err := json.Unmarshal([]byte(output), &events)
		testutil.NoError(t, err)
		testutil.Len(t, events, 0)
	})
}

func TestWeekCommand_Success(t *testing.T) {
	mock := &MockCalendarClient{
		ListEventsFunc: func(_ context.Context, _, _, _ string, _ int64) ([]*calendar.Event, error) {
			return []*calendar.Event{
				testutil.SampleEvent("week_event1"),
				testutil.SampleEvent("week_event2"),
			}, nil
		},
	}

	cmd := newWeekCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		// Should show events
		testutil.Contains(t, output, "Test Meeting")
	})
}

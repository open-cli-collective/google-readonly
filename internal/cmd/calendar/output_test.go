package calendar

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantJSON string
	}{
		{
			name: "single event",
			data: &calendar.Event{
				ID:      "event123",
				Summary: "Team Meeting",
			},
			wantJSON: `{
  "id": "event123",
  "summary": "Team Meeting",
  "start": null,
  "end": null,
  "status": "",
  "allDay": false
}
`,
		},
		{
			name: "event list",
			data: []*calendar.Event{
				{ID: "e1", Summary: "Event 1"},
				{ID: "e2", Summary: "Event 2"},
			},
		},
		{
			name: "calendar info",
			data: &calendar.CalendarInfo{
				ID:      "primary",
				Summary: "My Calendar",
				Primary: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := printJSON(tt.data)
			testutil.NoError(t, err)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()
			testutil.NotEmpty(t, output)

			// Verify it's valid JSON
			var parsed any
			err = json.Unmarshal([]byte(output), &parsed)
			testutil.NoError(t, err)

			if tt.wantJSON != "" {
				testutil.Equal(t, output, tt.wantJSON)
			}
		})
	}
}

func TestPrintEvent(t *testing.T) {
	tests := []struct {
		name            string
		event           *calendar.Event
		showDescription bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "basic event",
			event: &calendar.Event{
				ID:      "event123",
				Summary: "Team Meeting",
				Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:     &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
			},
			showDescription: false,
			wantContains: []string{
				"ID: event123",
				"Summary: Team Meeting",
				"When:",
			},
		},
		{
			name: "event with location",
			event: &calendar.Event{
				ID:       "event456",
				Summary:  "Offsite",
				Location: "123 Main St",
				Start:    &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:      &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
			},
			showDescription: false,
			wantContains: []string{
				"ID: event456",
				"Location: 123 Main St",
			},
		},
		{
			name: "event with hangout link",
			event: &calendar.Event{
				ID:          "event789",
				Summary:     "Video Call",
				HangoutLink: "https://meet.google.com/abc-xyz",
				Start:       &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:         &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
			},
			showDescription: false,
			wantContains: []string{
				"Meet: https://meet.google.com/abc-xyz",
			},
		},
		{
			name: "event with organizer display name",
			event: &calendar.Event{
				ID:      "event101",
				Summary: "Planning",
				Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:     &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
				Organizer: &calendar.Person{
					Email:       "alice@example.com",
					DisplayName: "Alice Smith",
				},
			},
			showDescription: false,
			wantContains: []string{
				"Organizer: Alice Smith <alice@example.com>",
			},
		},
		{
			name: "event with organizer email only",
			event: &calendar.Event{
				ID:      "event102",
				Summary: "Sync",
				Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:     &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
				Organizer: &calendar.Person{
					Email: "bob@example.com",
				},
			},
			showDescription: false,
			wantContains: []string{
				"Organizer: bob@example.com",
			},
			wantNotContains: []string{
				"<bob@example.com>", // Should not have angle brackets for email-only
			},
		},
		{
			name: "event with attendees",
			event: &calendar.Event{
				ID:      "event103",
				Summary: "Team Sync",
				Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:     &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
				Attendees: []calendar.Person{
					{Email: "alice@example.com", DisplayName: "Alice", Status: "accepted"},
					{Email: "bob@example.com", Status: "tentative"},
				},
			},
			showDescription: false,
			wantContains: []string{
				"Attendees: 2",
				"Alice <alice@example.com> (accepted)",
				"bob@example.com (tentative)",
			},
		},
		{
			name: "event with description shown",
			event: &calendar.Event{
				ID:          "event104",
				Summary:     "Meeting",
				Description: "Discuss project roadmap",
				Start:       &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:         &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
			},
			showDescription: true,
			wantContains: []string{
				"--- Description ---",
				"Discuss project roadmap",
			},
		},
		{
			name: "event with description hidden",
			event: &calendar.Event{
				ID:          "event105",
				Summary:     "Meeting",
				Description: "Secret notes",
				Start:       &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:         &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
			},
			showDescription: false,
			wantNotContains: []string{
				"--- Description ---",
				"Secret notes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printEvent(tt.event, tt.showDescription)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}
			for _, notWant := range tt.wantNotContains {
				testutil.NotContains(t, output, notWant)
			}
		})
	}
}

func TestPrintEventSummary(t *testing.T) {
	tests := []struct {
		name         string
		event        *calendar.Event
		wantContains []string
	}{
		{
			name: "basic summary",
			event: &calendar.Event{
				ID:      "event123",
				Summary: "Quick Sync",
				Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
				End:     &calendar.EventTime{DateTime: "2026-01-24T10:30:00Z"},
			},
			wantContains: []string{
				"ID: event123",
				"Summary: Quick Sync",
				"When:",
				"---",
			},
		},
		{
			name: "summary with location",
			event: &calendar.Event{
				ID:       "event456",
				Summary:  "Lunch",
				Location: "Cafe 42",
				Start:    &calendar.EventTime{DateTime: "2026-01-24T12:00:00Z"},
				End:      &calendar.EventTime{DateTime: "2026-01-24T13:00:00Z"},
			},
			wantContains: []string{
				"Location: Cafe 42",
			},
		},
		{
			name: "summary with hangout link",
			event: &calendar.Event{
				ID:          "event789",
				Summary:     "Video Chat",
				HangoutLink: "https://meet.google.com/xyz",
				Start:       &calendar.EventTime{DateTime: "2026-01-24T14:00:00Z"},
				End:         &calendar.EventTime{DateTime: "2026-01-24T15:00:00Z"},
			},
			wantContains: []string{
				"Meet: https://meet.google.com/xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printEventSummary(tt.event)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}
		})
	}
}

func TestPrintCalendar(t *testing.T) {
	tests := []struct {
		name         string
		cal          *calendar.CalendarInfo
		wantContains []string
	}{
		{
			name: "primary calendar",
			cal: &calendar.CalendarInfo{
				ID:         "primary",
				Summary:    "My Calendar",
				Primary:    true,
				AccessRole: "owner",
				TimeZone:   "America/Los_Angeles",
			},
			wantContains: []string{
				"ID: primary (primary)",
				"Name: My Calendar",
				"Access: owner",
				"Timezone: America/Los_Angeles",
				"---",
			},
		},
		{
			name: "secondary calendar with description",
			cal: &calendar.CalendarInfo{
				ID:          "work@group.calendar.google.com",
				Summary:     "Work Calendar",
				Description: "Team events and meetings",
				Primary:     false,
				AccessRole:  "writer",
				TimeZone:    "America/New_York",
			},
			wantContains: []string{
				"ID: work@group.calendar.google.com",
				"Name: Work Calendar",
				"Description: Team events and meetings",
				"Access: writer",
			},
		},
		{
			name: "calendar without description or timezone",
			cal: &calendar.CalendarInfo{
				ID:         "holidays@google.com",
				Summary:    "Holidays",
				Primary:    false,
				AccessRole: "reader",
			},
			wantContains: []string{
				"ID: holidays@google.com",
				"Name: Holidays",
				"Access: reader",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printCalendar(tt.cal)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}

			// Check that non-primary calendars don't have "(primary)"
			if !tt.cal.Primary {
				testutil.NotContains(t, output, "(primary)")
			}

			// Check that empty descriptions aren't printed
			if tt.cal.Description == "" {
				testutil.NotContains(t, output, "Description:")
			}

			// Check that empty timezones aren't printed
			if tt.cal.TimeZone == "" {
				testutil.NotContains(t, output, "Timezone:")
			}
		})
	}
}

func TestPrintCalendarNoPrimary(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCalendar(&calendar.CalendarInfo{
		ID:         "other@google.com",
		Summary:    "Other",
		Primary:    false,
		AccessRole: "reader",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()

	// Should have the ID without "(primary)"
	testutil.Contains(t, output, "ID: other@google.com")
	testutil.NotContains(t, output, "(primary)")
}

func TestPrintAttendeeWithoutStatus(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printEvent(&calendar.Event{
		ID:      "event123",
		Summary: "Meeting",
		Start:   &calendar.EventTime{DateTime: "2026-01-24T10:00:00Z"},
		End:     &calendar.EventTime{DateTime: "2026-01-24T11:00:00Z"},
		Attendees: []calendar.Person{
			{Email: "alice@example.com", DisplayName: "Alice"}, // No status
		},
	}, false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()

	// Should not have parentheses for status when status is empty
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Alice") {
			testutil.NotContains(t, line, "()")
			testutil.Contains(t, line, "Alice <alice@example.com>")
		}
	}
}

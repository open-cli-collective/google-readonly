package calendar

import (
	"strings"
	"testing"

	"google.golang.org/api/calendar/v3"
)

func TestParseEvent(t *testing.T) {
	t.Run("parses basic event", func(t *testing.T) {
		apiEvent := &calendar.Event{
			Id:          "event123",
			Summary:     "Team Meeting",
			Description: "Weekly sync",
			Location:    "Conference Room A",
			Status:      "confirmed",
			HtmlLink:    "https://calendar.google.com/event/123",
			Start: &calendar.EventDateTime{
				DateTime: "2026-01-24T10:00:00-05:00",
			},
			End: &calendar.EventDateTime{
				DateTime: "2026-01-24T11:00:00-05:00",
			},
		}

		event := ParseEvent(apiEvent)

		if got := event.ID; got != "event123" {
			t.Errorf("got %v, want %v", got, "event123")
		}
		if got := event.Summary; got != "Team Meeting" {
			t.Errorf("got %v, want %v", got, "Team Meeting")
		}
		if got := event.Description; got != "Weekly sync" {
			t.Errorf("got %v, want %v", got, "Weekly sync")
		}
		if got := event.Location; got != "Conference Room A" {
			t.Errorf("got %v, want %v", got, "Conference Room A")
		}
		if got := event.Status; got != "confirmed" {
			t.Errorf("got %v, want %v", got, "confirmed")
		}
		if event.AllDay {
			t.Error("got true, want false")
		}
	})

	t.Run("parses all-day event", func(t *testing.T) {
		apiEvent := &calendar.Event{
			Id:      "allday123",
			Summary: "Company Holiday",
			Start: &calendar.EventDateTime{
				Date: "2026-01-01",
			},
			End: &calendar.EventDateTime{
				Date: "2026-01-02",
			},
		}

		event := ParseEvent(apiEvent)

		if got := event.ID; got != "allday123" {
			t.Errorf("got %v, want %v", got, "allday123")
		}
		if !event.AllDay {
			t.Error("got false, want true")
		}
		if got := event.Start.Date; got != "2026-01-01" {
			t.Errorf("got %v, want %v", got, "2026-01-01")
		}
	})

	t.Run("parses event with organizer", func(t *testing.T) {
		apiEvent := &calendar.Event{
			Id:      "org123",
			Summary: "Project Review",
			Start: &calendar.EventDateTime{
				DateTime: "2026-01-24T14:00:00Z",
			},
			End: &calendar.EventDateTime{
				DateTime: "2026-01-24T15:00:00Z",
			},
			Organizer: &calendar.EventOrganizer{
				Email:       "boss@example.com",
				DisplayName: "The Boss",
				Self:        false,
			},
		}

		event := ParseEvent(apiEvent)

		if event.Organizer == nil {
			t.Fatal("expected non-nil, got nil")
		}
		if got := event.Organizer.Email; got != "boss@example.com" {
			t.Errorf("got %v, want %v", got, "boss@example.com")
		}
		if got := event.Organizer.DisplayName; got != "The Boss" {
			t.Errorf("got %v, want %v", got, "The Boss")
		}
	})

	t.Run("parses event with attendees", func(t *testing.T) {
		apiEvent := &calendar.Event{
			Id:      "att123",
			Summary: "Team Standup",
			Start: &calendar.EventDateTime{
				DateTime: "2026-01-24T09:00:00Z",
			},
			End: &calendar.EventDateTime{
				DateTime: "2026-01-24T09:15:00Z",
			},
			Attendees: []*calendar.EventAttendee{
				{
					Email:          "alice@example.com",
					DisplayName:    "Alice",
					ResponseStatus: "accepted",
				},
				{
					Email:          "bob@example.com",
					DisplayName:    "Bob",
					ResponseStatus: "tentative",
					Optional:       true,
				},
			},
		}

		event := ParseEvent(apiEvent)

		if len(event.Attendees) != 2 {
			t.Errorf("got length %d, want %d", len(event.Attendees), 2)
		}
		if got := event.Attendees[0].Email; got != "alice@example.com" {
			t.Errorf("got %v, want %v", got, "alice@example.com")
		}
		if got := event.Attendees[0].Status; got != "accepted" {
			t.Errorf("got %v, want %v", got, "accepted")
		}
		if got := event.Attendees[1].Email; got != "bob@example.com" {
			t.Errorf("got %v, want %v", got, "bob@example.com")
		}
		if !event.Attendees[1].Optional {
			t.Error("got false, want true")
		}
	})

	t.Run("handles event with hangout link", func(t *testing.T) {
		apiEvent := &calendar.Event{
			Id:          "meet123",
			Summary:     "Video Call",
			HangoutLink: "https://meet.google.com/abc-defg-hij",
			Start: &calendar.EventDateTime{
				DateTime: "2026-01-24T16:00:00Z",
			},
			End: &calendar.EventDateTime{
				DateTime: "2026-01-24T17:00:00Z",
			},
		}

		event := ParseEvent(apiEvent)

		if got := event.HangoutLink; got != "https://meet.google.com/abc-defg-hij" {
			t.Errorf("got %v, want %v", got, "https://meet.google.com/abc-defg-hij")
		}
	})
}

func TestParseCalendar(t *testing.T) {
	t.Run("parses calendar entry", func(t *testing.T) {
		apiCal := &calendar.CalendarListEntry{
			Id:          "primary",
			Summary:     "My Calendar",
			Description: "Personal calendar",
			Primary:     true,
			AccessRole:  "owner",
			TimeZone:    "America/New_York",
		}

		cal := ParseCalendar(apiCal)

		if got := cal.ID; got != "primary" {
			t.Errorf("got %v, want %v", got, "primary")
		}
		if got := cal.Summary; got != "My Calendar" {
			t.Errorf("got %v, want %v", got, "My Calendar")
		}
		if got := cal.Description; got != "Personal calendar" {
			t.Errorf("got %v, want %v", got, "Personal calendar")
		}
		if !cal.Primary {
			t.Error("got false, want true")
		}
		if got := cal.AccessRole; got != "owner" {
			t.Errorf("got %v, want %v", got, "owner")
		}
		if got := cal.TimeZone; got != "America/New_York" {
			t.Errorf("got %v, want %v", got, "America/New_York")
		}
	})

	t.Run("parses shared calendar", func(t *testing.T) {
		apiCal := &calendar.CalendarListEntry{
			Id:         "shared@group.calendar.google.com",
			Summary:    "Team Calendar",
			Primary:    false,
			AccessRole: "reader",
		}

		cal := ParseCalendar(apiCal)

		if cal.Primary {
			t.Error("got true, want false")
		}
		if got := cal.AccessRole; got != "reader" {
			t.Errorf("got %v, want %v", got, "reader")
		}
	})
}

func TestEventGetStartTime(t *testing.T) {
	t.Run("parses datetime", func(t *testing.T) {
		event := &Event{
			Start: &EventTime{
				DateTime: "2026-01-24T10:00:00-05:00",
			},
		}

		start, err := event.GetStartTime()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := start.Year(); got != 2026 {
			t.Errorf("got %v, want %v", got, 2026)
		}
		if got := int(start.Month()); got != 1 {
			t.Errorf("got %v, want %v", got, 1)
		}
		if got := start.Day(); got != 24 {
			t.Errorf("got %v, want %v", got, 24)
		}
	})

	t.Run("parses date for all-day event", func(t *testing.T) {
		event := &Event{
			AllDay: true,
			Start: &EventTime{
				Date: "2026-01-24",
			},
		}

		start, err := event.GetStartTime()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := start.Day(); got != 24 {
			t.Errorf("got %v, want %v", got, 24)
		}
	})

	t.Run("handles nil start", func(t *testing.T) {
		event := &Event{}

		start, err := event.GetStartTime()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !start.IsZero() {
			t.Error("got false, want true")
		}
	})
}

func TestEventFormatTimeRange(t *testing.T) {
	t.Run("formats same-day event", func(t *testing.T) {
		event := &Event{
			Start: &EventTime{
				DateTime: "2026-01-24T10:00:00-05:00",
			},
			End: &EventTime{
				DateTime: "2026-01-24T11:00:00-05:00",
			},
		}

		result := event.FormatTimeRange()
		if !strings.Contains(result, "Jan 24, 2026") {
			t.Errorf("expected %q to contain %q", result, "Jan 24, 2026")
		}
		if !strings.Contains(result, "10:00") {
			t.Errorf("expected %q to contain %q", result, "10:00")
		}
		if !strings.Contains(result, "11:00") {
			t.Errorf("expected %q to contain %q", result, "11:00")
		}
	})

	t.Run("formats all-day event", func(t *testing.T) {
		event := &Event{
			AllDay: true,
			Start: &EventTime{
				Date: "2026-01-24",
			},
			End: &EventTime{
				Date: "2026-01-25",
			},
		}

		result := event.FormatTimeRange()
		if !strings.Contains(result, "Jan 24, 2026") {
			t.Errorf("expected %q to contain %q", result, "Jan 24, 2026")
		}
		if !strings.Contains(result, "all day") {
			t.Errorf("expected %q to contain %q", result, "all day")
		}
	})
}

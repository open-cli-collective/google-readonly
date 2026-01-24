package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

		assert.Equal(t, "event123", event.ID)
		assert.Equal(t, "Team Meeting", event.Summary)
		assert.Equal(t, "Weekly sync", event.Description)
		assert.Equal(t, "Conference Room A", event.Location)
		assert.Equal(t, "confirmed", event.Status)
		assert.False(t, event.AllDay)
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

		assert.Equal(t, "allday123", event.ID)
		assert.True(t, event.AllDay)
		assert.Equal(t, "2026-01-01", event.Start.Date)
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

		assert.NotNil(t, event.Organizer)
		assert.Equal(t, "boss@example.com", event.Organizer.Email)
		assert.Equal(t, "The Boss", event.Organizer.DisplayName)
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

		assert.Len(t, event.Attendees, 2)
		assert.Equal(t, "alice@example.com", event.Attendees[0].Email)
		assert.Equal(t, "accepted", event.Attendees[0].Status)
		assert.Equal(t, "bob@example.com", event.Attendees[1].Email)
		assert.True(t, event.Attendees[1].Optional)
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

		assert.Equal(t, "https://meet.google.com/abc-defg-hij", event.HangoutLink)
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

		assert.Equal(t, "primary", cal.ID)
		assert.Equal(t, "My Calendar", cal.Summary)
		assert.Equal(t, "Personal calendar", cal.Description)
		assert.True(t, cal.Primary)
		assert.Equal(t, "owner", cal.AccessRole)
		assert.Equal(t, "America/New_York", cal.TimeZone)
	})

	t.Run("parses shared calendar", func(t *testing.T) {
		apiCal := &calendar.CalendarListEntry{
			Id:         "shared@group.calendar.google.com",
			Summary:    "Team Calendar",
			Primary:    false,
			AccessRole: "reader",
		}

		cal := ParseCalendar(apiCal)

		assert.False(t, cal.Primary)
		assert.Equal(t, "reader", cal.AccessRole)
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
		assert.NoError(t, err)
		assert.Equal(t, 2026, start.Year())
		assert.Equal(t, 1, int(start.Month()))
		assert.Equal(t, 24, start.Day())
	})

	t.Run("parses date for all-day event", func(t *testing.T) {
		event := &Event{
			AllDay: true,
			Start: &EventTime{
				Date: "2026-01-24",
			},
		}

		start, err := event.GetStartTime()
		assert.NoError(t, err)
		assert.Equal(t, 24, start.Day())
	})

	t.Run("handles nil start", func(t *testing.T) {
		event := &Event{}

		start, err := event.GetStartTime()
		assert.NoError(t, err)
		assert.True(t, start.IsZero())
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
		assert.Contains(t, result, "Jan 24, 2026")
		assert.Contains(t, result, "10:00")
		assert.Contains(t, result, "11:00")
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
		assert.Contains(t, result, "Jan 24, 2026")
		assert.Contains(t, result, "all day")
	})
}

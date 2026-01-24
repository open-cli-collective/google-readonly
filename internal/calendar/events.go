package calendar

import (
	"time"

	"google.golang.org/api/calendar/v3"
)

// Event represents a simplified calendar event for output
type Event struct {
	ID          string     `json:"id"`
	Summary     string     `json:"summary"`
	Description string     `json:"description,omitempty"`
	Location    string     `json:"location,omitempty"`
	Start       *EventTime `json:"start"`
	End         *EventTime `json:"end"`
	Status      string     `json:"status"`
	HTMLLink    string     `json:"htmlLink,omitempty"`
	HangoutLink string     `json:"hangoutLink,omitempty"`
	Organizer   *Person    `json:"organizer,omitempty"`
	Attendees   []Person   `json:"attendees,omitempty"`
	AllDay      bool       `json:"allDay"`
}

// EventTime represents a date or datetime
type EventTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

// Person represents an attendee or organizer
type Person struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Self        bool   `json:"self,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Status      string `json:"responseStatus,omitempty"`
}

// CalendarInfo represents a simplified calendar for output
type CalendarInfo struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Primary     bool   `json:"primary"`
	AccessRole  string `json:"accessRole"`
	TimeZone    string `json:"timeZone,omitempty"`
}

// ParseEvent converts a Google Calendar API event to our simplified Event
func ParseEvent(e *calendar.Event) *Event {
	event := &Event{
		ID:          e.Id,
		Summary:     e.Summary,
		Description: e.Description,
		Location:    e.Location,
		Status:      e.Status,
		HTMLLink:    e.HtmlLink,
		HangoutLink: e.HangoutLink,
	}

	// Parse start time
	if e.Start != nil {
		event.Start = &EventTime{
			DateTime: e.Start.DateTime,
			Date:     e.Start.Date,
			TimeZone: e.Start.TimeZone,
		}
		event.AllDay = e.Start.Date != "" && e.Start.DateTime == ""
	}

	// Parse end time
	if e.End != nil {
		event.End = &EventTime{
			DateTime: e.End.DateTime,
			Date:     e.End.Date,
			TimeZone: e.End.TimeZone,
		}
	}

	// Parse organizer
	if e.Organizer != nil {
		event.Organizer = &Person{
			Email:       e.Organizer.Email,
			DisplayName: e.Organizer.DisplayName,
			Self:        e.Organizer.Self,
		}
	}

	// Parse attendees
	if len(e.Attendees) > 0 {
		event.Attendees = make([]Person, len(e.Attendees))
		for i, a := range e.Attendees {
			event.Attendees[i] = Person{
				Email:       a.Email,
				DisplayName: a.DisplayName,
				Self:        a.Self,
				Optional:    a.Optional,
				Status:      a.ResponseStatus,
			}
		}
	}

	return event
}

// ParseCalendar converts a Google Calendar API calendar entry to our simplified CalendarInfo
func ParseCalendar(c *calendar.CalendarListEntry) *CalendarInfo {
	return &CalendarInfo{
		ID:          c.Id,
		Summary:     c.Summary,
		Description: c.Description,
		Primary:     c.Primary,
		AccessRole:  c.AccessRole,
		TimeZone:    c.TimeZone,
	}
}

// GetStartTime returns the event start time as a time.Time
func (e *Event) GetStartTime() (time.Time, error) {
	if e.Start == nil {
		return time.Time{}, nil
	}
	if e.Start.DateTime != "" {
		return time.Parse(time.RFC3339, e.Start.DateTime)
	}
	if e.Start.Date != "" {
		return time.Parse("2006-01-02", e.Start.Date)
	}
	return time.Time{}, nil
}

// GetEndTime returns the event end time as a time.Time
func (e *Event) GetEndTime() (time.Time, error) {
	if e.End == nil {
		return time.Time{}, nil
	}
	if e.End.DateTime != "" {
		return time.Parse(time.RFC3339, e.End.DateTime)
	}
	if e.End.Date != "" {
		return time.Parse("2006-01-02", e.End.Date)
	}
	return time.Time{}, nil
}

// FormatStartTime returns a human-readable start time string
func (e *Event) FormatStartTime() string {
	t, err := e.GetStartTime()
	if err != nil {
		return ""
	}
	if e.AllDay {
		return t.Format("Mon, Jan 2, 2006")
	}
	return t.Format("Mon, Jan 2, 2006 3:04 PM")
}

// FormatTimeRange returns a human-readable time range string
func (e *Event) FormatTimeRange() string {
	start, err := e.GetStartTime()
	if err != nil {
		return ""
	}
	end, err := e.GetEndTime()
	if err != nil {
		return e.FormatStartTime()
	}

	if e.AllDay {
		if start.Format("2006-01-02") == end.AddDate(0, 0, -1).Format("2006-01-02") {
			return start.Format("Mon, Jan 2, 2006") + " (all day)"
		}
		return start.Format("Mon, Jan 2") + " - " + end.AddDate(0, 0, -1).Format("Mon, Jan 2, 2006") + " (all day)"
	}

	if start.Format("2006-01-02") == end.Format("2006-01-02") {
		return start.Format("Mon, Jan 2, 2006 3:04 PM") + " - " + end.Format("3:04 PM")
	}
	return start.Format("Mon, Jan 2, 2006 3:04 PM") + " - " + end.Format("Mon, Jan 2, 2006 3:04 PM")
}

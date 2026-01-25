package calendar

import (
	"google.golang.org/api/calendar/v3"
)

// CalendarClientInterface defines the interface for Calendar client operations.
// This enables unit testing through mock implementations.
type CalendarClientInterface interface {
	// ListCalendars returns all calendars the user has access to
	ListCalendars() ([]*calendar.CalendarListEntry, error)

	// ListEvents returns events from the specified calendar within the given time range
	ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error)

	// GetEvent retrieves a single event by ID
	GetEvent(calendarID, eventID string) (*calendar.Event, error)
}

// Verify that Client implements CalendarClientInterface
var _ CalendarClientInterface = (*Client)(nil)

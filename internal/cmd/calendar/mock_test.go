package calendar

import (
	"google.golang.org/api/calendar/v3"
)

// MockCalendarClient is a configurable mock for CalendarClient.
type MockCalendarClient struct {
	ListCalendarsFunc func() ([]*calendar.CalendarListEntry, error)
	ListEventsFunc    func(calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error)
	GetEventFunc      func(calendarID, eventID string) (*calendar.Event, error)
}

// Verify MockCalendarClient implements CalendarClient
var _ CalendarClient = (*MockCalendarClient)(nil)

func (m *MockCalendarClient) ListCalendars() ([]*calendar.CalendarListEntry, error) {
	if m.ListCalendarsFunc != nil {
		return m.ListCalendarsFunc()
	}
	return nil, nil
}

func (m *MockCalendarClient) ListEvents(calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	if m.ListEventsFunc != nil {
		return m.ListEventsFunc(calendarID, timeMin, timeMax, maxResults)
	}
	return nil, nil
}

func (m *MockCalendarClient) GetEvent(calendarID, eventID string) (*calendar.Event, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(calendarID, eventID)
	}
	return nil, nil
}

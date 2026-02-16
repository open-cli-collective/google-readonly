package calendar

import (
	"context"

	"google.golang.org/api/calendar/v3"
)

// MockCalendarClient is a configurable mock for CalendarClient.
type MockCalendarClient struct {
	ListCalendarsFunc func(ctx context.Context) ([]*calendar.CalendarListEntry, error)
	ListEventsFunc    func(ctx context.Context, calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error)
	GetEventFunc      func(ctx context.Context, calendarID, eventID string) (*calendar.Event, error)
}

// Verify MockCalendarClient implements CalendarClient
var _ CalendarClient = (*MockCalendarClient)(nil)

func (m *MockCalendarClient) ListCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	if m.ListCalendarsFunc != nil {
		return m.ListCalendarsFunc(ctx)
	}
	return nil, nil
}

func (m *MockCalendarClient) ListEvents(ctx context.Context, calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	if m.ListEventsFunc != nil {
		return m.ListEventsFunc(ctx, calendarID, timeMin, timeMax, maxResults)
	}
	return nil, nil
}

func (m *MockCalendarClient) GetEvent(ctx context.Context, calendarID, eventID string) (*calendar.Event, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(ctx, calendarID, eventID)
	}
	return nil, nil
}

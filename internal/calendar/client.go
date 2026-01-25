package calendar

import (
	"context"
	"fmt"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Google Calendar API service
type Client struct {
	service *calendar.Service
}

// NewClient creates a new Calendar client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	client, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %w", err)
	}

	return &Client{
		service: srv,
	}, nil
}

// ListCalendars returns all calendars the user has access to
func (c *Client) ListCalendars() ([]*calendar.CalendarListEntry, error) {
	resp, err := c.service.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}
	return resp.Items, nil
}

// ListEvents returns events from the specified calendar within the given time range
func (c *Client) ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	call := c.service.Events.List(calendarID).
		SingleEvents(true).
		OrderBy("startTime")

	if timeMin != "" {
		call = call.TimeMin(timeMin)
	}
	if timeMax != "" {
		call = call.TimeMax(timeMax)
	}
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	return resp.Items, nil
}

// GetEvent retrieves a single event by ID
func (c *Client) GetEvent(calendarID, eventID string) (*calendar.Event, error) {
	event, err := c.service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	return event, nil
}

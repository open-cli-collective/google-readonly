// Package calendar provides a client for the Google Calendar API.
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
		return nil, fmt.Errorf("loading OAuth client: %w", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating Calendar service: %w", err)
	}

	return &Client{
		service: srv,
	}, nil
}

// ListCalendars returns all calendars the user has access to
func (c *Client) ListCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	resp, err := c.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing calendars: %w", err)
	}
	return resp.Items, nil
}

// ListEvents returns events from the specified calendar within the given time range
func (c *Client) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
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

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	return resp.Items, nil
}

// GetEvent retrieves a single event by ID
func (c *Client) GetEvent(ctx context.Context, calendarID, eventID string) (*calendar.Event, error) {
	event, err := c.service.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting event: %w", err)
	}
	return event, nil
}

// RSVPEvent updates the current user's RSVP status on an event.
// The response must be "accepted", "declined", or "tentative".
func (c *Client) RSVPEvent(ctx context.Context, calendarID, eventID, response string) error {
	event, err := c.service.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("getting event for RSVP: %w", err)
	}

	// Find the current user's attendee entry
	found := false
	for _, a := range event.Attendees {
		if a.Self {
			a.ResponseStatus = response
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("you are not listed as an attendee on this event")
	}

	_, err = c.service.Events.Patch(calendarID, eventID, &calendar.Event{
		Attendees: event.Attendees,
	}).SendUpdates("none").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("updating RSVP: %w", err)
	}
	return nil
}

// SetEventColor sets the color of a calendar event.
// The colorID must be a valid Google Calendar event color ID (1-11).
func (c *Client) SetEventColor(ctx context.Context, calendarID, eventID, colorID string) error {
	_, err := c.service.Events.Patch(calendarID, eventID, &calendar.Event{
		ColorId: colorID,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("setting event color: %w", err)
	}
	return nil
}

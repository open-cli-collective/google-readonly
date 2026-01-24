package calendar

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const (
	configDirName   = "google-readonly"
	credentialsFile = "credentials.json"
)

// Client wraps the Google Calendar API service
type Client struct {
	Service *calendar.Service
}

// NewClient creates a new Calendar client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	credPath := filepath.Join(configDir, credentialsFile)
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file at %s: %w\n\nPlease download your OAuth credentials from Google Cloud Console and save them to %s", credPath, err, credPath)
	}

	// Request read-only scopes for all supported services
	config, err := google.ConfigFromJSON(b,
		gmail.GmailReadonlyScope,
		calendar.CalendarReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	client, err := getHTTPClient(ctx, config)
	if err != nil {
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %w", err)
	}

	return &Client{
		Service: srv,
	}, nil
}

func getConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, configDirName)

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

func getHTTPClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	// Try to load token from keychain
	tok, err := keychain.GetToken()
	if err != nil {
		return nil, fmt.Errorf("no OAuth token found - please run 'gro init' first: %w", err)
	}

	// Create persistent token source that saves refreshed tokens
	tokenSource := keychain.NewPersistentTokenSource(config, tok)
	return oauth2.NewClient(ctx, tokenSource), nil
}

// ListCalendars returns all calendars the user has access to
func (c *Client) ListCalendars() ([]*calendar.CalendarListEntry, error) {
	resp, err := c.Service.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}
	return resp.Items, nil
}

// ListEvents returns events from the specified calendar within the given time range
func (c *Client) ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	call := c.Service.Events.List(calendarID).
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
	event, err := c.Service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	return event, nil
}

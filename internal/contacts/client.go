package contacts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const (
	configDirName   = "google-readonly"
	credentialsFile = "credentials.json"
)

// Client wraps the Google People API service for contacts
type Client struct {
	Service *people.Service
}

// NewClient creates a new Contacts client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	credPath := filepath.Join(configDir, credentialsFile)
	b, err := os.ReadFile(credPath)
	if err != nil {
		shortPath := gmail.ShortenPath(credPath)
		return nil, fmt.Errorf("unable to read credentials file at %s: %w\n\nPlease download your OAuth credentials from Google Cloud Console and save them to %s", shortPath, err, shortPath)
	}

	// Request read-only scope for contacts
	config, err := google.ConfigFromJSON(b, people.ContactsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	// Get token from keychain
	tok, err := keychain.GetToken()
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w\n\nRun 'gro init' to authenticate", err)
	}

	// Create persistent token source that saves refreshed tokens
	tokenSource := keychain.NewPersistentTokenSource(config, tok)

	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create People service: %w", err)
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

// ListContacts retrieves contacts from the user's account
func (c *Client) ListContacts(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
	call := c.Service.People.Connections.List("people/me").
		PersonFields("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,photos").
		PageSize(pageSize).
		SortOrder("LAST_NAME_ASCENDING")

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}

	return resp, nil
}

// SearchContacts searches for contacts matching a query
func (c *Client) SearchContacts(query string, pageSize int64) (*people.SearchResponse, error) {
	resp, err := c.Service.People.SearchContacts().
		Query(query).
		ReadMask("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,photos").
		PageSize(int64(pageSize)).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	return resp, nil
}

// GetContact retrieves a specific contact by resource name
func (c *Client) GetContact(resourceName string) (*people.Person, error) {
	resp, err := c.Service.People.Get(resourceName).
		PersonFields("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,urls,birthdays,events,relations,photos,metadata").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return resp, nil
}

// ListContactGroups retrieves all contact groups
func (c *Client) ListContactGroups(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
	call := c.Service.ContactGroups.List().
		PageSize(pageSize).
		GroupFields("name,groupType,memberCount")

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list contact groups: %w", err)
	}

	return resp, nil
}

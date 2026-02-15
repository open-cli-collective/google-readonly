package gmail

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Gmail API service
type Client struct {
	service      *gmail.Service
	userID       string
	labels       map[string]*gmail.Label
	labelsLoaded bool
	labelsMu     sync.RWMutex
}

// NewClient creates a new Gmail client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	client, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return &Client{
		service: srv,
		userID:  "me",
	}, nil
}

// FetchLabels retrieves and caches all labels from the Gmail account
func (c *Client) FetchLabels() error {
	// Check with read lock first to avoid unnecessary API calls
	c.labelsMu.RLock()
	if c.labelsLoaded {
		c.labelsMu.RUnlock()
		return nil
	}
	c.labelsMu.RUnlock()

	// Acquire write lock and check again (double-check locking)
	c.labelsMu.Lock()
	defer c.labelsMu.Unlock()

	// Re-check after acquiring write lock
	if c.labelsLoaded {
		return nil
	}

	resp, err := c.service.Users.Labels.List(c.userID).Do()
	if err != nil {
		return fmt.Errorf("fetching labels: %w", err)
	}

	c.labels = make(map[string]*gmail.Label)
	for _, label := range resp.Labels {
		c.labels[label.Id] = label
	}
	c.labelsLoaded = true

	return nil
}

// GetLabelName resolves a label ID to its display name
func (c *Client) GetLabelName(labelID string) string {
	c.labelsMu.RLock()
	defer c.labelsMu.RUnlock()

	if label, ok := c.labels[labelID]; ok {
		return label.Name
	}
	return labelID
}

// GetLabels returns all cached labels
func (c *Client) GetLabels() []*gmail.Label {
	c.labelsMu.RLock()
	defer c.labelsMu.RUnlock()

	if !c.labelsLoaded {
		return nil
	}
	labels := make([]*gmail.Label, 0, len(c.labels))
	for _, label := range c.labels {
		labels = append(labels, label)
	}
	return labels
}

// Profile represents a Gmail user profile.
type Profile struct {
	EmailAddress  string
	MessagesTotal int64
	ThreadsTotal  int64
}

// GetProfile retrieves the authenticated user's profile
func (c *Client) GetProfile() (*Profile, error) {
	profile, err := c.service.Users.GetProfile(c.userID).Do()
	if err != nil {
		return nil, fmt.Errorf("getting profile: %w", err)
	}
	return &Profile{
		EmailAddress:  profile.EmailAddress,
		MessagesTotal: profile.MessagesTotal,
		ThreadsTotal:  profile.ThreadsTotal,
	}, nil
}

// GetConfigDir returns the configuration directory path
// Deprecated: Use auth.GetConfigDir() instead
func GetConfigDir() (string, error) {
	return auth.GetConfigDir()
}

// GetCredentialsPath returns the path to credentials.json
// Deprecated: Use auth.GetCredentialsPath() instead
func GetCredentialsPath() (string, error) {
	return auth.GetCredentialsPath()
}

// GetOAuthConfig loads OAuth config from credentials file
// Deprecated: Use auth.GetOAuthConfig() instead
func GetOAuthConfig() (*oauth2.Config, error) {
	return auth.GetOAuthConfig()
}

// ExchangeAuthCode exchanges an authorization code for a token
// Deprecated: Use auth.ExchangeAuthCode() instead
func ExchangeAuthCode(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
	return auth.ExchangeAuthCode(ctx, config, code)
}

// GetAuthURL returns the OAuth authorization URL
// Deprecated: Use auth.GetAuthURL() instead
func GetAuthURL(config *oauth2.Config) string {
	return auth.GetAuthURL(config)
}

// ShortenPath replaces the home directory prefix with ~ for display purposes.
// Deprecated: Use auth.ShortenPath() instead
func ShortenPath(path string) string {
	return auth.ShortenPath(path)
}

package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const (
	configDirName   = "google-readonly"
	credentialsFile = "credentials.json"
	tokenFile       = "token.json"
)

// Client wraps the Gmail API service
type Client struct {
	Service      *gmail.Service
	UserID       string
	labels       map[string]*gmail.Label
	labelsLoaded bool
}

// NewClient creates a new Gmail client with OAuth2 authentication
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
		people.ContactsReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	client, err := getHTTPClient(ctx, config, configDir)
	if err != nil {
		return nil, err
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return &Client{
		Service: srv,
		UserID:  "me",
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

func getHTTPClient(ctx context.Context, config *oauth2.Config, configDir string) (*http.Client, error) {
	tokPath := filepath.Join(configDir, tokenFile)

	// Attempt migration from file to keychain (idempotent)
	_ = keychain.MigrateFromFile(tokPath)

	// Try to load token: keychain first, then file fallback
	tok, err := keychain.GetToken()
	if err != nil {
		tok, err = tokenFromFile(tokPath)
	}

	// No token found - need web auth flow
	if err != nil {
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		// Save new token to keychain (or file fallback)
		if kerr := keychain.SetToken(tok); kerr != nil {
			// Fall back to file storage
			if serr := saveToken(tokPath, tok); serr != nil {
				return nil, fmt.Errorf("failed to save token: %w", serr)
			}
		}
	}

	// Create persistent token source that saves refreshed tokens
	tokenSource := keychain.NewPersistentTokenSource(config, tok)
	return oauth2.NewClient(ctx, tokenSource), nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n\n%s\n\n", authURL)
	fmt.Print("Enter the authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange authorization code: %w", err)
	}
	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to save token: %w", err)
	}
	defer func() { _ = f.Close() }()
	return json.NewEncoder(f).Encode(token)
}

// FetchLabels retrieves and caches all labels from the Gmail account
func (c *Client) FetchLabels() error {
	if c.labelsLoaded {
		return nil
	}

	resp, err := c.Service.Users.Labels.List(c.UserID).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch labels: %w", err)
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
	if label, ok := c.labels[labelID]; ok {
		return label.Name
	}
	return labelID
}

// GetLabels returns all cached labels
func (c *Client) GetLabels() []*gmail.Label {
	if !c.labelsLoaded {
		return nil
	}
	labels := make([]*gmail.Label, 0, len(c.labels))
	for _, label := range c.labels {
		labels = append(labels, label)
	}
	return labels
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	return getConfigDir()
}

// GetCredentialsPath returns the path to credentials.json
func GetCredentialsPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, credentialsFile), nil
}

// GetOAuthConfig loads OAuth config from credentials file
func GetOAuthConfig() (*oauth2.Config, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}
	return google.ConfigFromJSON(b,
		gmail.GmailReadonlyScope,
		calendar.CalendarReadonlyScope,
		people.ContactsReadonlyScope,
	)
}

// ExchangeAuthCode exchanges an authorization code for a token
func ExchangeAuthCode(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
	return config.Exchange(ctx, code)
}

// GetAuthURL returns the OAuth authorization URL
func GetAuthURL(config *oauth2.Config) string {
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

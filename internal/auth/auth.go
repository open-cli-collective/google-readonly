package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

// AllScopes contains all OAuth scopes used by the application
var AllScopes = []string{
	gmail.GmailReadonlyScope,
	calendar.CalendarReadonlyScope,
	people.ContactsReadonlyScope,
	drive.DriveReadonlyScope,
}

// GetOAuthConfig loads OAuth configuration from credentials file with all scopes
func GetOAuthConfig() (*oauth2.Config, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}
	return google.ConfigFromJSON(b, AllScopes...)
}

// GetHTTPClient returns an HTTP client with OAuth2 authentication.
// It retrieves tokens from keychain (preferred) or falls back to file storage.
// Returns an error if no token is found - caller should direct user to run 'gro init'.
func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	config, err := GetOAuthConfig()
	if err != nil {
		return nil, err
	}

	// Try to load token from keychain
	tok, err := keychain.GetToken()
	if err != nil {
		// Try file fallback
		tokPath, pathErr := GetTokenPath()
		if pathErr != nil {
			return nil, fmt.Errorf("no OAuth token found - please run 'gro init' first: %w", err)
		}
		tok, err = tokenFromFile(tokPath)
		if err != nil {
			return nil, fmt.Errorf("no OAuth token found - please run 'gro init' first: %w", err)
		}
	}

	// Create persistent token source that saves refreshed tokens
	tokenSource := keychain.NewPersistentTokenSource(config, tok)
	return oauth2.NewClient(ctx, tokenSource), nil
}

// GetAuthURL returns the OAuth authorization URL for the given config
func GetAuthURL(config *oauth2.Config) string {
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeAuthCode exchanges an authorization code for a token
func ExchangeAuthCode(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
	return config.Exchange(ctx, code)
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

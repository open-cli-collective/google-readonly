// Package auth provides OAuth2 authentication and credential management for Google APIs.
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

// AllScopes contains all OAuth scopes used by the application.
// Gmail uses the modify scope for organizational operations (label, archive, star, mark read/unread).
// The modify scope is a superset of readonly — it includes all read access.
// Calendar uses both readonly (for calendar list metadata) and events (for RSVP/color operations).
var AllScopes = []string{
	gmail.GmailModifyScope,
	calendar.CalendarReadonlyScope,
	calendar.CalendarEventsScope,
	people.ContactsReadonlyScope,
	drive.DriveReadonlyScope,
}

// ScopeDescriptions maps OAuth scope URLs to human-friendly descriptions.
var ScopeDescriptions = map[string]string{
	gmail.GmailModifyScope:         "Gmail Modify — read messages, plus label, archive, star, and mark read/unread. No send or delete access.",
	gmail.GmailReadonlyScope:       "Gmail Read-Only — read messages and metadata.",
	calendar.CalendarReadonlyScope: "Calendar Read-Only — read calendars and events.",
	calendar.CalendarEventsScope:   "Calendar Events — read and update events (RSVP, color). No calendar settings access.",
	people.ContactsReadonlyScope:   "Contacts Read-Only — read contacts and groups.",
	drive.DriveReadonlyScope:       "Drive Read-Only — read files and metadata.",
}

// CheckScopesMigration compares the currently required scopes against the
// previously granted scopes. Returns a non-empty message if re-auth is needed.
func CheckScopesMigration(grantedScopes []string) string {
	if len(grantedScopes) == 0 {
		return ""
	}

	granted := make(map[string]bool, len(grantedScopes))
	for _, s := range grantedScopes {
		granted[s] = true
	}

	var missing []string
	for _, required := range AllScopes {
		if !granted[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) == 0 {
		return ""
	}

	msg := "This command requires additional permissions.\nYour current token only has read-only access.\n\nRun 'gro init' to re-authenticate with the updated scopes.\n\nNew scopes:\n"
	for _, s := range missing {
		desc := ScopeDescriptions[s]
		if desc == "" {
			desc = s
		}
		msg += "  - " + desc + "\n"
	}
	return msg
}

// GetOAuthConfig loads OAuth configuration from credentials file with all scopes
func GetOAuthConfig() (*oauth2.Config, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(credPath) //nolint:gosec // Path from user config directory
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
	tokenSource := keychain.NewPersistentTokenSource(ctx, config, tok)
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
	f, err := os.Open(file) //nolint:gosec // Path from user config directory
	if err != nil {
		return nil, err
	}
	// Close error is non-fatal for read operations
	defer func() { _ = f.Close() }()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

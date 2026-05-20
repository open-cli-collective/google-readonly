// Package auth provides OAuth2 authentication and credential management for Google APIs.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

// AllScopes contains all OAuth scopes used by the application.
// Gmail uses the modify scope for organizational operations (label, archive, star, mark read/unread).
// The modify scope is a superset of readonly — it includes all read access.
// Calendar uses both readonly (for calendar list metadata) and events (for RSVP/color operations).
// Note: calendar.events also permits event creation, which is an accepted trade-off for RSVP/color support.
// The architecture test (TestNoDestructiveAPIMethodsInProductionCode) prevents accidental misuse.
// Contacts uses the full contacts scope for group management and starring.
// The contacts scope is a superset of contacts.readonly — it includes all read access.
// Profile is required for people/me (names, emailAddresses fields) used by `gro me` and init verification.
var AllScopes = []string{
	gmail.GmailModifyScope,
	calendar.CalendarReadonlyScope,
	calendar.CalendarEventsScope,
	people.ContactsScope,
	people.UserinfoProfileScope,
	drive.DriveReadonlyScope,
	drive.DriveMetadataScope,
}

// ScopeDescriptions maps OAuth scope URLs to human-friendly descriptions.
var ScopeDescriptions = map[string]string{
	gmail.GmailModifyScope:         "Gmail Modify — read messages, plus label, archive, star, and mark read/unread. No send or delete access.",
	gmail.GmailReadonlyScope:       "Gmail Read-Only — read messages and metadata.",
	calendar.CalendarReadonlyScope: "Calendar Read-Only — read calendars and events.",
	calendar.CalendarEventsScope:   "Calendar Events — read and update events (RSVP, color). No calendar settings access.",
	people.ContactsScope:           "Contacts — read contacts and groups, plus manage group membership and starring.",
	people.ContactsReadonlyScope:   "Contacts Read-Only — read contacts and groups.",
	people.UserinfoProfileScope:    "Profile — read the authenticated user's name and email address (required for 'gro me').",
	drive.DriveReadonlyScope:       "Drive Read-Only — read files and metadata.",
	drive.DriveMetadataScope:       "Drive Metadata — read and update file metadata (star/unstar). No file content write access.",
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

// GetOAuthConfig loads the OAuth client config from the deployment-material
// OAuth client JSON referenced by config.yml's oauth_client_path (§1.2 — not
// a secret; lives on disk, never the keyring), with all scopes.
func GetOAuthConfig() (*oauth2.Config, error) {
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return nil, err
	}
	path := config.ExpandPath(cfg.OAuthClientPath)
	b, err := os.ReadFile(path) //nolint:gosec // deployment-material path from config
	if err != nil {
		return nil, fmt.Errorf("unable to read OAuth client JSON %s (run 'gro init'): %w",
			config.ShortenPath(path), err)
	}
	return google.ConfigFromJSON(b, AllScopes...)
}

// GetHTTPClient returns an HTTP client with OAuth2 authentication. The token
// is read solely from the OS keyring via credstore (§1.1/§2.3 — no
// security/secret-tool shell-out, no token.json fallback). The active
// credential_ref is captured once here; refreshed tokens persist back to that
// exact ref via the closure passed to the token source (the sole sanctioned
// non-ingress keyring write). Returns an actionable error if no token exists.
func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	oauthCfg, err := GetOAuthConfig()
	if err != nil {
		return nil, err
	}

	st, err := keychain.Open()
	if err != nil {
		return nil, err
	}
	tok, err := st.Token()
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("no OAuth token found - please run 'gro init' first: %w", err)
	}
	ref := st.Ref()
	_ = st.Close() // do not hold the Store for the client's lifetime

	persist := func(t *oauth2.Token) error {
		ps, perr := keychain.OpenRef(ref) // runMigration=false: refresh is not ingress
		if perr != nil {
			return perr
		}
		defer func() { _ = ps.Close() }()
		return ps.SetToken(t)
	}

	tokenSource := keychain.NewPersistentTokenSource(ctx, oauthCfg, tok, persist)
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

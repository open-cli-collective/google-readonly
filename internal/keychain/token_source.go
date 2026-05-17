package keychain

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

// TokenPersister persists a refreshed OAuth token. It is supplied by the
// caller (auth.GetHTTPClient) bound to the ref captured at construction time,
// so the only sanctioned non-ingress keyring write (runtime token refresh,
// standard §ix / line 174) updates the existing oauth_token key under the
// active ref — never a new key, never a different ref.
type TokenPersister func(*oauth2.Token) error

// PersistentTokenSource wraps a TokenSource and persists refreshed tokens via
// the injected persister. This solves the problem where oauth2's automatic
// token refresh does not persist the rotated token back to storage.
type PersistentTokenSource struct {
	mu      sync.Mutex
	base    oauth2.TokenSource
	current *oauth2.Token
	persist TokenPersister
}

// NewPersistentTokenSource creates a TokenSource that persists refreshed
// tokens through persist. When the underlying oauth2 package refreshes an
// expired token, this wrapper detects the change and writes it back via the
// caller-captured persister (no long-lived credstore Store handle).
func NewPersistentTokenSource(ctx context.Context, cfg *oauth2.Config, initial *oauth2.Token, persist TokenPersister) oauth2.TokenSource {
	return &PersistentTokenSource{
		base:    cfg.TokenSource(ctx, initial),
		current: initial,
		persist: persist,
	}
}

// Token returns a valid token, refreshing and persisting if necessary.
// This method is safe for concurrent use.
func (p *PersistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := p.base.Token() // may trigger refresh
	if err != nil {
		return nil, err
	}

	// Refresh occurred (access token rotated): persist the new token.
	if p.current == nil || token.AccessToken != p.current.AccessToken {
		if p.persist != nil {
			if err := p.persist(token); err != nil {
				// Non-fatal: the token is still valid in memory. The user
				// re-authenticates next run only if this keeps failing.
				fmt.Fprintf(os.Stderr, "Warning: failed to persist refreshed token: %v\n", err)
			}
		}
		p.current = token
	}

	return token, nil
}

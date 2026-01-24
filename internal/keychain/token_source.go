package keychain

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

// PersistentTokenSource wraps a TokenSource and persists refreshed tokens.
// This solves the problem where oauth2's automatic token refresh doesn't
// persist the new token to storage.
type PersistentTokenSource struct {
	mu      sync.Mutex
	base    oauth2.TokenSource
	current *oauth2.Token
}

// NewPersistentTokenSource creates a TokenSource that persists refreshed tokens.
// When the underlying oauth2 package refreshes an expired token, this wrapper
// detects the change and saves the new token to secure storage.
func NewPersistentTokenSource(config *oauth2.Config, initial *oauth2.Token) oauth2.TokenSource {
	// Create base token source that handles refresh
	base := config.TokenSource(context.Background(), initial)

	return &PersistentTokenSource{
		base:    base,
		current: initial,
	}
}

// Token returns a valid token, refreshing and persisting if necessary.
// This method is safe for concurrent use.
func (p *PersistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get token (may trigger refresh)
	token, err := p.base.Token()
	if err != nil {
		return nil, err
	}

	// Check if token changed (refresh occurred)
	if p.current == nil || token.AccessToken != p.current.AccessToken {
		// Persist the new token
		if err := SetToken(token); err != nil {
			// Log warning but don't fail - token is still valid in memory
			// User will need to re-auth on next run if this persists
			fmt.Fprintf(os.Stderr, "Warning: failed to persist refreshed token: %v\n", err)
		}
		p.current = token
	}

	return token, nil
}

// Package people provides a client for the Google People API focused on the
// authenticated user's own profile (people/me). It is separate from
// internal/contacts so the cmd/me command and the cmd/contacts command can
// evolve independently.
package people

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	peopleapi "google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Profile is the subset of People API data we surface for `gro me`.
type Profile struct {
	ResourceName string
	DisplayName  string
	PrimaryEmail string
}

// Client wraps the Google People API service for the authenticated user.
type Client struct {
	service *peopleapi.Service
}

// NewClient creates a new People client with OAuth2 authentication.
func NewClient(ctx context.Context) (*Client, error) {
	httpClient, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading OAuth client: %w", err)
	}

	srv, err := peopleapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("creating People service: %w", err)
	}

	return &Client{service: srv}, nil
}

// GetMe returns the authenticated user's profile via people/me.
func (c *Client) GetMe(ctx context.Context) (*Profile, error) {
	person, err := c.service.People.Get("people/me").
		PersonFields("names,emailAddresses").
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	return &Profile{
		ResourceName: person.ResourceName,
		DisplayName:  pickPrimaryName(person),
		PrimaryEmail: pickPrimaryEmail(person),
	}, nil
}

// pickPrimaryName returns the metadata-primary display name, falling back to
// the first entry if no entry is marked primary.
func pickPrimaryName(p *peopleapi.Person) string {
	if p == nil {
		return ""
	}
	for _, n := range p.Names {
		if n.Metadata != nil && n.Metadata.Primary {
			return n.DisplayName
		}
	}
	if len(p.Names) > 0 {
		return p.Names[0].DisplayName
	}
	return ""
}

// pickPrimaryEmail returns the metadata-primary email, falling back to the
// first entry if no entry is marked primary.
func pickPrimaryEmail(p *peopleapi.Person) string {
	if p == nil {
		return ""
	}
	for _, e := range p.EmailAddresses {
		if e.Metadata != nil && e.Metadata.Primary {
			return e.Value
		}
	}
	if len(p.EmailAddresses) > 0 {
		return p.EmailAddresses[0].Value
	}
	return ""
}

// IsInsufficientScopeError reports whether err is a 403 caused by missing
// OAuth scopes (as opposed to a 403 caused by API-not-enabled,
// project-not-permitted, etc.). Distinguishing these matters because only
// scope errors should suggest `gro init` re-auth.
func IsInsufficientScopeError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Code != 403 {
		return false
	}
	for _, e := range apiErr.Errors {
		if e.Reason == "insufficientPermissions" || e.Reason == "ACCESS_TOKEN_SCOPE_INSUFFICIENT" {
			return true
		}
	}
	msg := strings.ToLower(apiErr.Message)
	if strings.Contains(msg, "insufficient authentication scopes") ||
		strings.Contains(msg, "access_token_scope_insufficient") {
		return true
	}
	return false
}

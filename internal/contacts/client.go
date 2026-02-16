// Package contacts provides a client for the Google People API.
package contacts

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

// Client wraps the Google People API service for contacts
type Client struct {
	service *people.Service
}

// NewClient creates a new Contacts client with OAuth2 authentication
func NewClient(ctx context.Context) (*Client, error) {
	client, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading OAuth client: %w", err)
	}

	srv, err := people.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating People service: %w", err)
	}

	return &Client{
		service: srv,
	}, nil
}

// ListContacts retrieves contacts from the user's account
func (c *Client) ListContacts(ctx context.Context, pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
	call := c.service.People.Connections.List("people/me").
		PersonFields("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,photos").
		PageSize(pageSize).
		SortOrder("LAST_NAME_ASCENDING")

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}

	return resp, nil
}

// SearchContacts searches for contacts matching a query
func (c *Client) SearchContacts(ctx context.Context, query string, pageSize int64) (*people.SearchResponse, error) {
	resp, err := c.service.People.SearchContacts().
		Query(query).
		ReadMask("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,photos").
		PageSize(int64(pageSize)).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("searching contacts: %w", err)
	}

	return resp, nil
}

// GetContact retrieves a specific contact by resource name
func (c *Client) GetContact(ctx context.Context, resourceName string) (*people.Person, error) {
	resp, err := c.service.People.Get(resourceName).
		PersonFields("names,emailAddresses,phoneNumbers,organizations,addresses,biographies,urls,birthdays,events,relations,photos,metadata").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("getting contact: %w", err)
	}

	return resp, nil
}

// ListContactGroups retrieves all contact groups
func (c *Client) ListContactGroups(ctx context.Context, pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
	call := c.service.ContactGroups.List().
		PageSize(pageSize).
		GroupFields("name,groupType,memberCount")

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing contact groups: %w", err)
	}

	return resp, nil
}

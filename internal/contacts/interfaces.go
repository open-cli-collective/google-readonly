package contacts

import (
	"google.golang.org/api/people/v1"
)

// ContactsClientInterface defines the interface for Contacts client operations.
// This enables unit testing through mock implementations.
type ContactsClientInterface interface {
	// ListContacts retrieves contacts from the user's account
	ListContacts(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error)

	// SearchContacts searches for contacts matching a query
	SearchContacts(query string, pageSize int64) (*people.SearchResponse, error)

	// GetContact retrieves a specific contact by resource name
	GetContact(resourceName string) (*people.Person, error)

	// ListContactGroups retrieves all contact groups
	ListContactGroups(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error)
}

// Verify that Client implements ContactsClientInterface
var _ ContactsClientInterface = (*Client)(nil)

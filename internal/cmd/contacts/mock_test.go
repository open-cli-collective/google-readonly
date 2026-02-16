package contacts

import (
	"context"

	"google.golang.org/api/people/v1"
)

// MockContactsClient is a configurable mock for ContactsClient.
type MockContactsClient struct {
	ListContactsFunc      func(ctx context.Context, pageToken string, pageSize int64) (*people.ListConnectionsResponse, error)
	SearchContactsFunc    func(ctx context.Context, query string, pageSize int64) (*people.SearchResponse, error)
	GetContactFunc        func(ctx context.Context, resourceName string) (*people.Person, error)
	ListContactGroupsFunc func(ctx context.Context, pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error)
}

// Verify MockContactsClient implements ContactsClient
var _ ContactsClient = (*MockContactsClient)(nil)

func (m *MockContactsClient) ListContacts(ctx context.Context, pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
	if m.ListContactsFunc != nil {
		return m.ListContactsFunc(ctx, pageToken, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) SearchContacts(ctx context.Context, query string, pageSize int64) (*people.SearchResponse, error) {
	if m.SearchContactsFunc != nil {
		return m.SearchContactsFunc(ctx, query, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) GetContact(ctx context.Context, resourceName string) (*people.Person, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(ctx, resourceName)
	}
	return nil, nil
}

func (m *MockContactsClient) ListContactGroups(ctx context.Context, pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
	if m.ListContactGroupsFunc != nil {
		return m.ListContactGroupsFunc(ctx, pageToken, pageSize)
	}
	return nil, nil
}

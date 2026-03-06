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
	AddToGroupFunc        func(ctx context.Context, groupResourceName string, contactResourceNames []string) error
	RemoveFromGroupFunc   func(ctx context.Context, groupResourceName string, contactResourceNames []string) error
	ResolveGroupNameFunc  func(ctx context.Context, name string) (string, error)
	SearchContactIDsFunc  func(ctx context.Context, query string, pageSize int64) ([]string, error)
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

func (m *MockContactsClient) AddToGroup(ctx context.Context, groupResourceName string, contactResourceNames []string) error {
	if m.AddToGroupFunc != nil {
		return m.AddToGroupFunc(ctx, groupResourceName, contactResourceNames)
	}
	return nil
}

func (m *MockContactsClient) RemoveFromGroup(ctx context.Context, groupResourceName string, contactResourceNames []string) error {
	if m.RemoveFromGroupFunc != nil {
		return m.RemoveFromGroupFunc(ctx, groupResourceName, contactResourceNames)
	}
	return nil
}

func (m *MockContactsClient) ResolveGroupName(ctx context.Context, name string) (string, error) {
	if m.ResolveGroupNameFunc != nil {
		return m.ResolveGroupNameFunc(ctx, name)
	}
	return "", nil
}

func (m *MockContactsClient) SearchContactIDs(ctx context.Context, query string, pageSize int64) ([]string, error) {
	if m.SearchContactIDsFunc != nil {
		return m.SearchContactIDsFunc(ctx, query, pageSize)
	}
	return nil, nil
}

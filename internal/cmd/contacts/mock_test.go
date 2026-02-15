package contacts

import (
	"google.golang.org/api/people/v1"
)

// MockContactsClient is a configurable mock for ContactsClient.
type MockContactsClient struct {
	ListContactsFunc      func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error)
	SearchContactsFunc    func(query string, pageSize int64) (*people.SearchResponse, error)
	GetContactFunc        func(resourceName string) (*people.Person, error)
	ListContactGroupsFunc func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error)
}

// Verify MockContactsClient implements ContactsClient
var _ ContactsClient = (*MockContactsClient)(nil)

func (m *MockContactsClient) ListContacts(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
	if m.ListContactsFunc != nil {
		return m.ListContactsFunc(pageToken, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) SearchContacts(query string, pageSize int64) (*people.SearchResponse, error) {
	if m.SearchContactsFunc != nil {
		return m.SearchContactsFunc(query, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) GetContact(resourceName string) (*people.Person, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(resourceName)
	}
	return nil, nil
}

func (m *MockContactsClient) ListContactGroups(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
	if m.ListContactGroupsFunc != nil {
		return m.ListContactGroupsFunc(pageToken, pageSize)
	}
	return nil, nil
}

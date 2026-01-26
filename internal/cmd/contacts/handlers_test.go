package contacts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/people/v1"

	contactsapi "github.com/open-cli-collective/google-readonly/internal/contacts"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// captureOutput captures stdout during test execution
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// withMockClient sets up a mock client factory for tests
func withMockClient(mock contactsapi.ContactsClientInterface, f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (contactsapi.ContactsClientInterface, error) {
		return mock, nil
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

// withFailingClientFactory sets up a factory that returns an error
func withFailingClientFactory(f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (contactsapi.ContactsClientInterface, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

func TestListCommand_Success(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactsFunc: func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
			return &people.ListConnectionsResponse{
				Connections: []*people.Person{
					testutil.SamplePerson("people/c123"),
					testutil.SamplePerson("people/c456"),
				},
			}, nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "people/c123")
		assert.Contains(t, output, "John Doe")
		assert.Contains(t, output, "2 contact(s)")
	})
}

func TestListCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactsFunc: func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
			return &people.ListConnectionsResponse{
				Connections: []*people.Person{
					testutil.SamplePerson("people/c123"),
				},
			}, nil
		},
	}

	cmd := newListCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var contacts []*contactsapi.Contact
		err := json.Unmarshal([]byte(output), &contacts)
		assert.NoError(t, err)
		assert.Len(t, contacts, 1)
	})
}

func TestListCommand_Empty(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactsFunc: func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
			return &people.ListConnectionsResponse{
				Connections: []*people.Person{},
			}, nil
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No contacts found")
	})
}

func TestListCommand_APIError(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactsFunc: func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newListCommand()

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list contacts")
	})
}

func TestListCommand_ClientCreationError(t *testing.T) {
	cmd := newListCommand()

	withFailingClientFactory(func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Contacts client")
	})
}

func TestSearchCommand_Success(t *testing.T) {
	mock := &testutil.MockContactsClient{
		SearchContactsFunc: func(query string, pageSize int64) (*people.SearchResponse, error) {
			assert.Equal(t, "John", query)
			return &people.SearchResponse{
				Results: []*people.SearchResult{
					{Person: testutil.SamplePerson("people/c123")},
				},
			}, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"John"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "John Doe")
		assert.Contains(t, output, "1 contact(s)")
	})
}

func TestSearchCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockContactsClient{
		SearchContactsFunc: func(query string, pageSize int64) (*people.SearchResponse, error) {
			return &people.SearchResponse{
				Results: []*people.SearchResult{
					{Person: testutil.SamplePerson("people/c123")},
				},
			}, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"John", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var contacts []*contactsapi.Contact
		err := json.Unmarshal([]byte(output), &contacts)
		assert.NoError(t, err)
		assert.Len(t, contacts, 1)
	})
}

func TestSearchCommand_NoResults(t *testing.T) {
	mock := &testutil.MockContactsClient{
		SearchContactsFunc: func(query string, pageSize int64) (*people.SearchResponse, error) {
			return &people.SearchResponse{
				Results: []*people.SearchResult{},
			}, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No contacts found")
	})
}

func TestSearchCommand_APIError(t *testing.T) {
	mock := &testutil.MockContactsClient{
		SearchContactsFunc: func(query string, pageSize int64) (*people.SearchResponse, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"John"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to search contacts")
	})
}

func TestGetCommand_Success(t *testing.T) {
	mock := &testutil.MockContactsClient{
		GetContactFunc: func(resourceName string) (*people.Person, error) {
			assert.Equal(t, "people/c123", resourceName)
			return testutil.SamplePerson("people/c123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"people/c123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "people/c123")
		assert.Contains(t, output, "John Doe")
		assert.Contains(t, output, "john@example.com")
	})
}

func TestGetCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockContactsClient{
		GetContactFunc: func(resourceName string) (*people.Person, error) {
			return testutil.SamplePerson("people/c123"), nil
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"people/c123", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var contact contactsapi.Contact
		err := json.Unmarshal([]byte(output), &contact)
		assert.NoError(t, err)
		assert.Equal(t, "people/c123", contact.ResourceName)
	})
}

func TestGetCommand_NotFound(t *testing.T) {
	mock := &testutil.MockContactsClient{
		GetContactFunc: func(resourceName string) (*people.Person, error) {
			return nil, errors.New("contact not found")
		},
	}

	cmd := newGetCommand()
	cmd.SetArgs([]string{"people/nonexistent"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get contact")
	})
}

func TestGroupsCommand_Success(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactGroupsFunc: func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
			return &people.ListContactGroupsResponse{
				ContactGroups: []*people.ContactGroup{
					{
						ResourceName: "contactGroups/123",
						Name:         "Friends",
						GroupType:    "USER_CONTACT_GROUP",
						MemberCount:  5,
					},
					{
						ResourceName: "contactGroups/456",
						Name:         "Family",
						GroupType:    "USER_CONTACT_GROUP",
						MemberCount:  10,
					},
				},
			}, nil
		},
	}

	cmd := newGroupsCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Friends")
		assert.Contains(t, output, "Family")
		assert.Contains(t, output, "2 contact group(s)")
	})
}

func TestGroupsCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactGroupsFunc: func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
			return &people.ListContactGroupsResponse{
				ContactGroups: []*people.ContactGroup{
					{
						ResourceName: "contactGroups/123",
						Name:         "Friends",
						GroupType:    "USER_CONTACT_GROUP",
						MemberCount:  5,
					},
				},
			}, nil
		},
	}

	cmd := newGroupsCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var groups []*contactsapi.ContactGroup
		err := json.Unmarshal([]byte(output), &groups)
		assert.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.Equal(t, "Friends", groups[0].Name)
	})
}

func TestGroupsCommand_Empty(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactGroupsFunc: func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
			return &people.ListContactGroupsResponse{
				ContactGroups: []*people.ContactGroup{},
			}, nil
		},
	}

	cmd := newGroupsCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No contact groups found")
	})
}

func TestGroupsCommand_APIError(t *testing.T) {
	mock := &testutil.MockContactsClient{
		ListContactGroupsFunc: func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
			return nil, errors.New("API error")
		},
	}

	cmd := newGroupsCommand()

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list contact groups")
	})
}

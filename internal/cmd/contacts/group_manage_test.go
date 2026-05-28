package contacts

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestAddToGroupCommand_Success(t *testing.T) {
	var addedGroup string
	var addedContacts []string

	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, name string) (string, error) {
			testutil.Equal(t, name, "Friends")
			return "contactGroups/abc123", nil
		},
		AddToGroupFunc: func(_ context.Context, groupResourceName string, contactResourceNames []string) error {
			addedGroup = groupResourceName
			addedContacts = contactResourceNames
			return nil
		},
	}

	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111", "people/c222"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, addedGroup, "contactGroups/abc123")
		testutil.Len(t, addedContacts, 2)
		testutil.Contains(t, output, "Added to group 'Friends' 2 contact(s)")
	})
}

func TestAddToGroupCommand_DryRun(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
	}

	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run]")
		testutil.Contains(t, output, "add to group 'Friends'")
		testutil.Contains(t, output, "1 contact(s)")
	})
}

func TestAddToGroupCommand_GroupNotFound(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("group not found: Nonexistent")
		},
	}

	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Nonexistent", "people/c111"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "resolving group")
	})
}

func TestAddToGroupCommand_APIError(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
		AddToGroupFunc: func(_ context.Context, _ string, _ []string) error {
			return errors.New("API error")
		},
	}

	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "adding contacts to group")
	})
}

func TestAddToGroupCommand_ClientError(t *testing.T) {
	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Contacts client")
	})
}

func TestAddToGroupCommand_Query(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
		SearchContactIDsFunc: func(_ context.Context, query string, _ int64) ([]string, error) {
			testutil.Equal(t, query, "John")
			return []string{"people/c111", "people/c222"}, nil
		},
		AddToGroupFunc: func(_ context.Context, _ string, contacts []string) error {
			testutil.Len(t, contacts, 2)
			return nil
		},
	}

	cmd := newAddToGroupCommand()
	cmd.SetArgs([]string{"Friends", "--query", "John"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "2 contact(s)")
	})
}

func TestRemoveFromGroupCommand_Success(t *testing.T) {
	var removedGroup string
	var removedContacts []string

	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
		RemoveFromGroupFunc: func(_ context.Context, groupResourceName string, contactResourceNames []string) error {
			removedGroup = groupResourceName
			removedContacts = contactResourceNames
			return nil
		},
	}

	cmd := newRemoveFromGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, removedGroup, "contactGroups/abc123")
		testutil.Len(t, removedContacts, 1)
		testutil.Contains(t, output, "Removed from group 'Friends' 1 contact(s)")
	})
}

func TestRemoveFromGroupCommand_DryRun(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
	}

	cmd := newRemoveFromGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run]")
		testutil.Contains(t, output, "remove from group 'Friends'")
	})
}

func TestRemoveFromGroupCommand_APIError(t *testing.T) {
	mock := &MockContactsClient{
		ResolveGroupNameFunc: func(_ context.Context, _ string) (string, error) {
			return "contactGroups/abc123", nil
		},
		RemoveFromGroupFunc: func(_ context.Context, _ string, _ []string) error {
			return errors.New("API error")
		},
	}

	cmd := newRemoveFromGroupCommand()
	cmd.SetArgs([]string{"Friends", "people/c111"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "removing contacts from group")
	})
}

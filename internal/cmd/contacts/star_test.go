package contacts

import (
	"context"
	"errors"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestStarCommand_Success(t *testing.T) {
	var addedGroup string
	var addedContacts []string

	mock := &MockContactsClient{
		AddToGroupFunc: func(_ context.Context, groupResourceName string, contactResourceNames []string) error {
			addedGroup = groupResourceName
			addedContacts = contactResourceNames
			return nil
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"people/c111", "people/c222"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, addedGroup, "contactGroups/starred")
		testutil.Len(t, addedContacts, 2)
		testutil.Contains(t, output, "Starred 2 contact(s)")
	})
}

func TestStarCommand_DryRun(t *testing.T) {
	mock := &MockContactsClient{}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"people/c111", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run]")
		testutil.Contains(t, output, "star")
		testutil.Contains(t, output, "1 contact(s)")
	})
}

func TestStarCommand_Query(t *testing.T) {
	mock := &MockContactsClient{
		SearchContactIDsFunc: func(_ context.Context, query string, _ int64) ([]string, error) {
			testutil.Equal(t, query, "John")
			return []string{"people/c111"}, nil
		},
		AddToGroupFunc: func(_ context.Context, group string, _ []string) error {
			testutil.Equal(t, group, "contactGroups/starred")
			return nil
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"--query", "John"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Starred 1 contact(s)")
	})
}

func TestStarCommand_APIError(t *testing.T) {
	mock := &MockContactsClient{
		AddToGroupFunc: func(_ context.Context, _ string, _ []string) error {
			return errors.New("API error")
		},
	}

	cmd := newStarCommand()
	cmd.SetArgs([]string{"people/c111"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "starring contacts")
	})
}

func TestStarCommand_ClientError(t *testing.T) {
	cmd := newStarCommand()
	cmd.SetArgs([]string{"people/c111"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Contacts client")
	})
}

func TestUnstarCommand_Success(t *testing.T) {
	var removedGroup string

	mock := &MockContactsClient{
		RemoveFromGroupFunc: func(_ context.Context, groupResourceName string, _ []string) error {
			removedGroup = groupResourceName
			return nil
		},
	}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"people/c111"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Equal(t, removedGroup, "contactGroups/starred")
		testutil.Contains(t, output, "Unstarred 1 contact(s)")
	})
}

func TestUnstarCommand_DryRun(t *testing.T) {
	mock := &MockContactsClient{}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"people/c111", "--dry-run"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "[dry-run]")
		testutil.Contains(t, output, "unstar")
	})
}

func TestUnstarCommand_APIError(t *testing.T) {
	mock := &MockContactsClient{
		RemoveFromGroupFunc: func(_ context.Context, _ string, _ []string) error {
			return errors.New("API error")
		},
	}

	cmd := newUnstarCommand()
	cmd.SetArgs([]string{"people/c111"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "unstarring contacts")
	})
}

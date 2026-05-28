package mail

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// withMockClient sets up a mock client factory for tests
func withMockClient(mock MailClient, f func()) {
	testutil.WithFactory(&ClientFactory, func(_ context.Context) (MailClient, error) {
		return mock, nil
	}, f)
}

// withFailingClientFactory sets up a factory that returns an error
func withFailingClientFactory(f func()) {
	testutil.WithFactory(&ClientFactory, func(_ context.Context) (MailClient, error) {
		return nil, errors.New("connection failed")
	}, f)
}

func TestSearchCommand_Success(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, query string, maxResults int64) ([]*gmailapi.Message, int, error) {
			testutil.Equal(t, query, "is:unread")
			testutil.Equal(t, maxResults, int64(10))
			return testutil.SampleMessages(2), 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		// Verify output contains expected message data
		testutil.Contains(t, output, "ID: msg_a")
		testutil.Contains(t, output, "ID: msg_b")
		testutil.Contains(t, output, "Test Subject")
	})
}

func TestSearchCommand_NoResults(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, _ string, _ int64) ([]*gmailapi.Message, int, error) {
			return []*gmailapi.Message{}, 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "No messages found")
	})
}

func TestSearchCommand_APIError(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, _ string, _ int64) ([]*gmailapi.Message, int, error) {
			return nil, 0, errors.New("API quota exceeded")
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "searching messages")
	})
}

func TestSearchCommand_ClientCreationError(t *testing.T) {
	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating Gmail client")
	})
}

func TestSearchCommand_IDsOutput(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessageIDsFunc: func(_ context.Context, query string, _ int64) ([]string, error) {
			testutil.Equal(t, query, "is:inbox")
			return []string{"msg1", "msg2", "msg3"}, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:inbox", "--ids"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "msg1")
		testutil.Contains(t, output, "msg2")
		testutil.Contains(t, output, "msg3")
	})
}

func TestSearchCommand_SkippedMessages(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, _ string, _ int64) ([]*gmailapi.Message, int, error) {
			return testutil.SampleMessages(2), 3, nil // 3 messages skipped
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "3 message(s) could not be retrieved")
	})
}

func TestReadCommand_Success(t *testing.T) {
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, messageID string, includeBody bool) (*gmailapi.Message, error) {
			testutil.Equal(t, messageID, "msg123")
			testutil.True(t, includeBody)
			return testutil.SampleMessage("msg123"), nil
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "ID: msg123")
		testutil.Contains(t, output, "Test Subject")
		testutil.Contains(t, output, "--- Body ---")
	})
}

func TestReadCommand_NotFound(t *testing.T) {
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			return nil, errors.New("message not found")
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "reading message")
	})
}

func TestThreadCommand_Success(t *testing.T) {
	mock := &MockGmailClient{
		GetThreadFunc: func(_ context.Context, id string) ([]*gmailapi.Message, error) {
			testutil.Equal(t, id, "thread123")
			return testutil.SampleMessages(3), nil
		},
	}

	cmd := newThreadCommand()
	cmd.SetArgs([]string{"thread123"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "Thread contains 3 message(s)")
		testutil.Contains(t, output, "Message 1 of 3")
		testutil.Contains(t, output, "Message 2 of 3")
		testutil.Contains(t, output, "Message 3 of 3")
	})
}

func TestLabelsCommand_Success(t *testing.T) {
	mock := &MockGmailClient{
		FetchLabelsFunc: func(_ context.Context) error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return testutil.SampleLabels()
		},
	}

	cmd := newLabelsCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "NAME")
		testutil.Contains(t, output, "TYPE")
		testutil.Contains(t, output, "Work")
		testutil.Contains(t, output, "user")
	})
}

func TestLabelsCommand_Empty(t *testing.T) {
	mock := &MockGmailClient{
		FetchLabelsFunc: func(_ context.Context) error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return []*gmail.Label{}
		},
	}

	cmd := newLabelsCommand()

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "No labels found")
	})
}

func TestListAttachmentsCommand_Success(t *testing.T) {
	mock := &MockGmailClient{
		GetAttachmentsFunc: func(_ context.Context, _ string) ([]*gmailapi.Attachment, error) {
			return []*gmailapi.Attachment{
				testutil.SampleAttachment("report.pdf"),
				testutil.SampleAttachment("data.xlsx"),
			}, nil
		},
	}

	cmd := newListAttachmentsCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "2 attachment(s)")
		testutil.Contains(t, output, "report.pdf")
		testutil.Contains(t, output, "data.xlsx")
	})
}

func TestListAttachmentsCommand_NoAttachments(t *testing.T) {
	mock := &MockGmailClient{
		GetAttachmentsFunc: func(_ context.Context, _ string) ([]*gmailapi.Attachment, error) {
			return []*gmailapi.Attachment{}, nil
		},
	}

	cmd := newListAttachmentsCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		testutil.Contains(t, output, "No attachments found")
	})
}

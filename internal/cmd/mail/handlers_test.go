package mail

import (
	"context"
	"encoding/json"
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

func TestSearchCommand_JSONOutput(t *testing.T) {
	mock := &MockGmailClient{
		SearchMessagesFunc: func(_ context.Context, _ string, _ int64) ([]*gmailapi.Message, int, error) {
			return testutil.SampleMessages(1), 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread", "--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		// Verify JSON output is valid
		var messages []*gmailapi.Message
		err := json.Unmarshal([]byte(output), &messages)
		testutil.NoError(t, err)
		testutil.Len(t, messages, 1)
		testutil.Equal(t, messages[0].ID, "msg_a")
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

func TestReadCommand_JSONOutput(t *testing.T) {
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			return testutil.SampleMessage("msg123"), nil
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"msg123", "--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var msg gmailapi.Message
		err := json.Unmarshal([]byte(output), &msg)
		testutil.NoError(t, err)
		testutil.Equal(t, msg.ID, "msg123")
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

func TestThreadCommand_JSONOutput(t *testing.T) {
	mock := &MockGmailClient{
		GetThreadFunc: func(_ context.Context, _ string) ([]*gmailapi.Message, error) {
			return testutil.SampleMessages(2), nil
		},
	}

	cmd := newThreadCommand()
	cmd.SetArgs([]string{"thread123", "--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var messages []*gmailapi.Message
		err := json.Unmarshal([]byte(output), &messages)
		testutil.NoError(t, err)
		testutil.Len(t, messages, 2)
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

func TestLabelsCommand_JSONOutput(t *testing.T) {
	mock := &MockGmailClient{
		FetchLabelsFunc: func(_ context.Context) error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return testutil.SampleLabels()
		},
	}

	cmd := newLabelsCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})

		var labels []Label
		err := json.Unmarshal([]byte(output), &labels)
		testutil.NoError(t, err)
		testutil.Greater(t, len(labels), 0)
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

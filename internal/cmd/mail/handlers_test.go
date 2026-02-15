package mail

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
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
func withMockClient(mock gmailapi.GmailClientInterface, f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (gmailapi.GmailClientInterface, error) {
		return mock, nil
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

// withFailingClientFactory sets up a factory that returns an error
func withFailingClientFactory(f func()) {
	originalFactory := ClientFactory
	ClientFactory = func() (gmailapi.GmailClientInterface, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { ClientFactory = originalFactory }()
	f()
}

func TestSearchCommand_Success(t *testing.T) {
	mock := &testutil.MockGmailClient{
		SearchMessagesFunc: func(query string, maxResults int64) ([]*gmailapi.Message, int, error) {
			assert.Equal(t, "is:unread", query)
			assert.Equal(t, int64(10), maxResults)
			return testutil.SampleMessages(2), 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		// Verify output contains expected message data
		assert.Contains(t, output, "ID: msg_a")
		assert.Contains(t, output, "ID: msg_b")
		assert.Contains(t, output, "Test Subject")
	})
}

func TestSearchCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockGmailClient{
		SearchMessagesFunc: func(_ string, _ int64) ([]*gmailapi.Message, int, error) {
			return testutil.SampleMessages(1), 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		// Verify JSON output is valid
		var messages []*gmailapi.Message
		err := json.Unmarshal([]byte(output), &messages)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "msg_a", messages[0].ID)
	})
}

func TestSearchCommand_NoResults(t *testing.T) {
	mock := &testutil.MockGmailClient{
		SearchMessagesFunc: func(_ string, _ int64) ([]*gmailapi.Message, int, error) {
			return []*gmailapi.Message{}, 0, nil
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No messages found")
	})
}

func TestSearchCommand_APIError(t *testing.T) {
	mock := &testutil.MockGmailClient{
		SearchMessagesFunc: func(_ string, _ int64) ([]*gmailapi.Message, int, error) {
			return nil, 0, errors.New("API quota exceeded")
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to search messages")
	})
}

func TestSearchCommand_ClientCreationError(t *testing.T) {
	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withFailingClientFactory(func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Gmail client")
	})
}

func TestSearchCommand_SkippedMessages(t *testing.T) {
	mock := &testutil.MockGmailClient{
		SearchMessagesFunc: func(_ string, _ int64) ([]*gmailapi.Message, int, error) {
			return testutil.SampleMessages(2), 3, nil // 3 messages skipped
		},
	}

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"is:unread"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "3 message(s) could not be retrieved")
	})
}

func TestReadCommand_Success(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetMessageFunc: func(messageID string, includeBody bool) (*gmailapi.Message, error) {
			assert.Equal(t, "msg123", messageID)
			assert.True(t, includeBody)
			return testutil.SampleMessage("msg123"), nil
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "ID: msg123")
		assert.Contains(t, output, "Test Subject")
		assert.Contains(t, output, "--- Body ---")
	})
}

func TestReadCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetMessageFunc: func(_ string, _ bool) (*gmailapi.Message, error) {
			return testutil.SampleMessage("msg123"), nil
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"msg123", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var msg gmailapi.Message
		err := json.Unmarshal([]byte(output), &msg)
		assert.NoError(t, err)
		assert.Equal(t, "msg123", msg.ID)
	})
}

func TestReadCommand_NotFound(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetMessageFunc: func(_ string, _ bool) (*gmailapi.Message, error) {
			return nil, errors.New("message not found")
		},
	}

	cmd := newReadCommand()
	cmd.SetArgs([]string{"nonexistent"})

	withMockClient(mock, func() {
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read message")
	})
}

func TestThreadCommand_Success(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetThreadFunc: func(id string) ([]*gmailapi.Message, error) {
			assert.Equal(t, "thread123", id)
			return testutil.SampleMessages(3), nil
		},
	}

	cmd := newThreadCommand()
	cmd.SetArgs([]string{"thread123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Thread contains 3 message(s)")
		assert.Contains(t, output, "Message 1 of 3")
		assert.Contains(t, output, "Message 2 of 3")
		assert.Contains(t, output, "Message 3 of 3")
	})
}

func TestThreadCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetThreadFunc: func(_ string) ([]*gmailapi.Message, error) {
			return testutil.SampleMessages(2), nil
		},
	}

	cmd := newThreadCommand()
	cmd.SetArgs([]string{"thread123", "--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var messages []*gmailapi.Message
		err := json.Unmarshal([]byte(output), &messages)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
	})
}

func TestLabelsCommand_Success(t *testing.T) {
	mock := &testutil.MockGmailClient{
		FetchLabelsFunc: func() error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return testutil.SampleLabels()
		},
	}

	cmd := newLabelsCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "TYPE")
		assert.Contains(t, output, "Work")
		assert.Contains(t, output, "user")
	})
}

func TestLabelsCommand_JSONOutput(t *testing.T) {
	mock := &testutil.MockGmailClient{
		FetchLabelsFunc: func() error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return testutil.SampleLabels()
		},
	}

	cmd := newLabelsCommand()
	cmd.SetArgs([]string{"--json"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		var labels []Label
		err := json.Unmarshal([]byte(output), &labels)
		assert.NoError(t, err)
		assert.Greater(t, len(labels), 0)
	})
}

func TestLabelsCommand_Empty(t *testing.T) {
	mock := &testutil.MockGmailClient{
		FetchLabelsFunc: func() error {
			return nil
		},
		GetLabelsFunc: func() []*gmail.Label {
			return []*gmail.Label{}
		},
	}

	cmd := newLabelsCommand()

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No labels found")
	})
}

func TestListAttachmentsCommand_Success(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetAttachmentsFunc: func(_ string) ([]*gmailapi.Attachment, error) {
			return []*gmailapi.Attachment{
				testutil.SampleAttachment("report.pdf"),
				testutil.SampleAttachment("data.xlsx"),
			}, nil
		},
	}

	cmd := newListAttachmentsCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "2 attachment(s)")
		assert.Contains(t, output, "report.pdf")
		assert.Contains(t, output, "data.xlsx")
	})
}

func TestListAttachmentsCommand_NoAttachments(t *testing.T) {
	mock := &testutil.MockGmailClient{
		GetAttachmentsFunc: func(_ string) ([]*gmailapi.Attachment, error) {
			return []*gmailapi.Attachment{}, nil
		},
	}

	cmd := newListAttachmentsCommand()
	cmd.SetArgs([]string{"msg123"})

	withMockClient(mock, func() {
		output := captureOutput(t, func() {
			err := cmd.Execute()
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No attachments found")
	})
}

package mail

import (
	"google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
)

// MockGmailClient is a configurable mock for MailClient.
// Set the function fields to control behavior in tests.
type MockGmailClient struct {
	GetMessageFunc               func(messageID string, includeBody bool) (*gmailapi.Message, error)
	SearchMessagesFunc           func(query string, maxResults int64) ([]*gmailapi.Message, int, error)
	GetThreadFunc                func(id string) ([]*gmailapi.Message, error)
	FetchLabelsFunc              func() error
	GetLabelNameFunc             func(labelID string) string
	GetLabelsFunc                func() []*gmail.Label
	GetAttachmentsFunc           func(messageID string) ([]*gmailapi.Attachment, error)
	DownloadAttachmentFunc       func(messageID, attachmentID string) ([]byte, error)
	DownloadInlineAttachmentFunc func(messageID, partID string) ([]byte, error)
	GetProfileFunc               func() (*gmailapi.Profile, error)
}

// Verify MockGmailClient implements MailClient
var _ MailClient = (*MockGmailClient)(nil)

func (m *MockGmailClient) GetMessage(messageID string, includeBody bool) (*gmailapi.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(messageID, includeBody)
	}
	return nil, nil
}

func (m *MockGmailClient) SearchMessages(query string, maxResults int64) ([]*gmailapi.Message, int, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(query, maxResults)
	}
	return nil, 0, nil
}

func (m *MockGmailClient) GetThread(id string) ([]*gmailapi.Message, error) {
	if m.GetThreadFunc != nil {
		return m.GetThreadFunc(id)
	}
	return nil, nil
}

func (m *MockGmailClient) FetchLabels() error {
	if m.FetchLabelsFunc != nil {
		return m.FetchLabelsFunc()
	}
	return nil
}

func (m *MockGmailClient) GetLabelName(labelID string) string {
	if m.GetLabelNameFunc != nil {
		return m.GetLabelNameFunc(labelID)
	}
	return labelID
}

func (m *MockGmailClient) GetLabels() []*gmail.Label {
	if m.GetLabelsFunc != nil {
		return m.GetLabelsFunc()
	}
	return nil
}

func (m *MockGmailClient) GetAttachments(messageID string) ([]*gmailapi.Attachment, error) {
	if m.GetAttachmentsFunc != nil {
		return m.GetAttachmentsFunc(messageID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadAttachment(messageID, attachmentID string) ([]byte, error) {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(messageID, attachmentID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadInlineAttachment(messageID, partID string) ([]byte, error) {
	if m.DownloadInlineAttachmentFunc != nil {
		return m.DownloadInlineAttachmentFunc(messageID, partID)
	}
	return nil, nil
}

func (m *MockGmailClient) GetProfile() (*gmailapi.Profile, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc()
	}
	return nil, nil
}

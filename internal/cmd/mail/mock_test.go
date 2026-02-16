package mail

import (
	"context"

	"google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
)

// MockGmailClient is a configurable mock for MailClient.
// Set the function fields to control behavior in tests.
type MockGmailClient struct {
	GetMessageFunc               func(ctx context.Context, messageID string, includeBody bool) (*gmailapi.Message, error)
	SearchMessagesFunc           func(ctx context.Context, query string, maxResults int64) ([]*gmailapi.Message, int, error)
	GetThreadFunc                func(ctx context.Context, id string) ([]*gmailapi.Message, error)
	FetchLabelsFunc              func(ctx context.Context) error
	GetLabelNameFunc             func(labelID string) string
	GetLabelsFunc                func() []*gmail.Label
	GetAttachmentsFunc           func(ctx context.Context, messageID string) ([]*gmailapi.Attachment, error)
	DownloadAttachmentFunc       func(ctx context.Context, messageID, attachmentID string) ([]byte, error)
	DownloadInlineAttachmentFunc func(ctx context.Context, messageID, partID string) ([]byte, error)
	GetProfileFunc               func(ctx context.Context) (*gmailapi.Profile, error)
}

// Verify MockGmailClient implements MailClient
var _ MailClient = (*MockGmailClient)(nil)

func (m *MockGmailClient) GetMessage(ctx context.Context, messageID string, includeBody bool) (*gmailapi.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, messageID, includeBody)
	}
	return nil, nil
}

func (m *MockGmailClient) SearchMessages(ctx context.Context, query string, maxResults int64) ([]*gmailapi.Message, int, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(ctx, query, maxResults)
	}
	return nil, 0, nil
}

func (m *MockGmailClient) GetThread(ctx context.Context, id string) ([]*gmailapi.Message, error) {
	if m.GetThreadFunc != nil {
		return m.GetThreadFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockGmailClient) FetchLabels(ctx context.Context) error {
	if m.FetchLabelsFunc != nil {
		return m.FetchLabelsFunc(ctx)
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

func (m *MockGmailClient) GetAttachments(ctx context.Context, messageID string) ([]*gmailapi.Attachment, error) {
	if m.GetAttachmentsFunc != nil {
		return m.GetAttachmentsFunc(ctx, messageID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error) {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(ctx, messageID, attachmentID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadInlineAttachment(ctx context.Context, messageID, partID string) ([]byte, error) {
	if m.DownloadInlineAttachmentFunc != nil {
		return m.DownloadInlineAttachmentFunc(ctx, messageID, partID)
	}
	return nil, nil
}

func (m *MockGmailClient) GetProfile(ctx context.Context) (*gmailapi.Profile, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc(ctx)
	}
	return nil, nil
}

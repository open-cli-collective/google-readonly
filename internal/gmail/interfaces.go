package gmail

import (
	"google.golang.org/api/gmail/v1"
)

// GmailClientInterface defines the interface for Gmail client operations.
// This enables unit testing through mock implementations.
type GmailClientInterface interface {
	// GetMessage retrieves a single message by ID
	GetMessage(messageID string, includeBody bool) (*Message, error)

	// SearchMessages searches for messages matching the query
	SearchMessages(query string, maxResults int64) ([]*Message, int, error)

	// GetThread retrieves all messages in a thread
	GetThread(id string) ([]*Message, error)

	// FetchLabels retrieves and caches all labels from the Gmail account
	FetchLabels() error

	// GetLabelName resolves a label ID to its display name
	GetLabelName(labelID string) string

	// GetLabels returns all cached labels
	GetLabels() []*gmail.Label

	// GetAttachments retrieves attachment metadata for a message
	GetAttachments(messageID string) ([]*Attachment, error)

	// DownloadAttachment downloads a single attachment by message ID and attachment ID
	DownloadAttachment(messageID string, attachmentID string) ([]byte, error)

	// DownloadInlineAttachment downloads an attachment that has inline data
	DownloadInlineAttachment(messageID string, partID string) ([]byte, error)
}

// Verify that Client implements GmailClientInterface
var _ GmailClientInterface = (*Client)(nil)

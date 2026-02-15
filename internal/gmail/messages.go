package gmail

import (
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-readonly/internal/log"
)

// Message represents a simplified email message
type Message struct {
	ID          string        `json:"id"`
	ThreadID    string        `json:"threadId"`
	Subject     string        `json:"subject"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	Date        string        `json:"date"`
	Snippet     string        `json:"snippet"`
	Body        string        `json:"body,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`
	Labels      []string      `json:"labels,omitempty"`
	Categories  []string      `json:"categories,omitempty"`
}

// Attachment represents metadata about an email attachment
type Attachment struct {
	Filename     string `json:"filename"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	AttachmentID string `json:"attachmentId,omitempty"`
	PartID       string `json:"partId"`
	IsInline     bool   `json:"isInline"`
}

// SearchMessages searches for messages matching the query.
// Returns messages, the count of messages that failed to fetch, and any error.
func (c *Client) SearchMessages(query string, maxResults int64) ([]*Message, int, error) {
	call := c.service.Users.Messages.List(c.userID).Q(query)
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search messages: %w", err)
	}

	var messages []*Message
	var skipped int
	for _, msg := range resp.Messages {
		m, err := c.GetMessage(msg.Id, false)
		if err != nil {
			skipped++
			log.Debug("skipped message %s: %v", msg.Id, err)
			continue
		}
		messages = append(messages, m)
	}

	if skipped > 0 {
		log.Warn("skipped %d message(s) due to fetch errors (use -v for details)", skipped)
	}

	return messages, skipped, nil
}

// GetMessage retrieves a single message by ID
func (c *Client) GetMessage(messageID string, includeBody bool) (*Message, error) {
	format := "metadata"
	if includeBody {
		format = "full"
	}

	// Fetch labels for resolution
	if err := c.FetchLabels(); err != nil {
		return nil, err
	}

	msg, err := c.service.Users.Messages.Get(c.userID, messageID).Format(format).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return parseMessage(msg, includeBody, c.GetLabelName), nil
}

// GetThread retrieves all messages in a thread.
// The id parameter can be either a thread ID or a message ID.
// If a message ID is provided, the thread ID is resolved automatically.
func (c *Client) GetThread(id string) ([]*Message, error) {
	// Fetch labels for resolution
	if err := c.FetchLabels(); err != nil {
		return nil, err
	}

	thread, err := c.service.Users.Threads.Get(c.userID, id).Format("full").Do()
	if err != nil {
		// If the ID wasn't found as a thread ID, try treating it as a message ID
		msg, msgErr := c.service.Users.Messages.Get(c.userID, id).Format("minimal").Do()
		if msgErr != nil {
			// Return the original thread error if message lookup also fails
			return nil, fmt.Errorf("failed to get thread: %w", err)
		}
		// Use the thread ID from the message
		thread, err = c.service.Users.Threads.Get(c.userID, msg.ThreadId).Format("full").Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get thread: %w", err)
		}
	}

	var messages []*Message
	for _, msg := range thread.Messages {
		messages = append(messages, parseMessage(msg, true, c.GetLabelName))
	}

	return messages, nil
}

// LabelResolver is a function that resolves a label ID to its display name
type LabelResolver func(labelID string) string

func parseMessage(msg *gmail.Message, includeBody bool, resolver LabelResolver) *Message {
	m := &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
	}

	// Extract labels and categories (doesn't need Payload)
	m.Labels, m.Categories = extractLabelsAndCategories(msg.LabelIds, resolver)

	// Early return if Payload is nil
	if msg.Payload == nil {
		return m
	}

	// Extract headers
	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			m.Subject = header.Value
		case "from":
			m.From = header.Value
		case "to":
			m.To = header.Value
		case "date":
			m.Date = header.Value
		}
	}

	if includeBody {
		m.Body = extractBody(msg.Payload)
		m.Attachments = extractAttachments(msg.Payload, "")
	}

	return m
}

// extractLabelsAndCategories separates label IDs into user labels and Gmail categories
func extractLabelsAndCategories(labelIDs []string, resolver LabelResolver) ([]string, []string) {
	var labels, categories []string

	// System labels to exclude from display
	systemLabels := map[string]bool{
		"INBOX": true, "SENT": true, "DRAFT": true, "SPAM": true,
		"TRASH": true, "UNREAD": true, "STARRED": true, "IMPORTANT": true,
		"CHAT": true, "CATEGORY_PERSONAL": true,
	}

	for _, labelID := range labelIDs {
		// Check if it's a category
		if strings.HasPrefix(labelID, "CATEGORY_") {
			// Convert CATEGORY_UPDATES -> updates
			category := strings.ToLower(strings.TrimPrefix(labelID, "CATEGORY_"))
			if category != "personal" { // Skip CATEGORY_PERSONAL (default)
				categories = append(categories, category)
			}
			continue
		}

		// Skip system labels
		if systemLabels[labelID] {
			continue
		}

		// Resolve user labels to their display names
		if resolver != nil {
			labels = append(labels, resolver(labelID))
		} else {
			labels = append(labels, labelID)
		}
	}

	return labels, categories
}

// extractAttachments recursively traverses message parts to find attachments
func extractAttachments(payload *gmail.MessagePart, partPath string) []*Attachment {
	var attachments []*Attachment

	// Check if this part is an attachment
	if isAttachment(payload) {
		att := &Attachment{
			Filename: payload.Filename,
			MimeType: payload.MimeType,
			PartID:   partPath,
			IsInline: isInlineAttachment(payload),
		}
		if payload.Body != nil {
			att.Size = payload.Body.Size
			att.AttachmentID = payload.Body.AttachmentId
		}
		attachments = append(attachments, att)
	}

	// Recursively check nested parts
	for i, part := range payload.Parts {
		childPath := fmt.Sprintf("%d", i)
		if partPath != "" {
			childPath = fmt.Sprintf("%s.%d", partPath, i)
		}
		attachments = append(attachments, extractAttachments(part, childPath)...)
	}

	return attachments
}

// isAttachment determines if a message part is an attachment
func isAttachment(part *gmail.MessagePart) bool {
	// Primary check: has a filename
	if part.Filename != "" {
		return true
	}

	// Secondary check: Content-Disposition header indicates attachment
	for _, header := range part.Headers {
		if strings.ToLower(header.Name) == "content-disposition" {
			value := strings.ToLower(header.Value)
			if strings.HasPrefix(value, "attachment") {
				return true
			}
		}
	}

	return false
}

// isInlineAttachment checks if attachment is inline (e.g., embedded image)
func isInlineAttachment(part *gmail.MessagePart) bool {
	for _, header := range part.Headers {
		if strings.ToLower(header.Name) == "content-disposition" {
			return strings.HasPrefix(strings.ToLower(header.Value), "inline")
		}
	}
	return false
}

func extractBody(payload *gmail.MessagePart) string {
	// Try plain text first, then fall back to HTML
	for _, mimeType := range []string{"text/plain", "text/html"} {
		if body := findBodyByMimeType(payload, mimeType); body != "" {
			return body
		}
	}
	return ""
}

// findBodyByMimeType searches for body content matching the given MIME type
func findBodyByMimeType(part *gmail.MessagePart, mimeType string) string {
	// Check current part
	if part.MimeType == mimeType && part.Body != nil && part.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
			return string(decoded)
		}
	}

	// Check nested parts recursively
	for _, child := range part.Parts {
		if body := findBodyByMimeType(child, mimeType); body != "" {
			return body
		}
	}

	return ""
}

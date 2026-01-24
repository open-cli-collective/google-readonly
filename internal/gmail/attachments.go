package gmail

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// GetAttachments retrieves attachment metadata for a message
func (c *Client) GetAttachments(messageID string) ([]*Attachment, error) {
	msg, err := c.Service.Users.Messages.Get(c.UserID, messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return extractAttachments(msg.Payload, ""), nil
}

// DownloadAttachment downloads a single attachment by message ID and attachment ID
func (c *Client) DownloadAttachment(messageID string, attachmentID string) ([]byte, error) {
	att, err := c.Service.Users.Messages.Attachments.Get(c.UserID, messageID, attachmentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to download attachment: %w", err)
	}

	data, err := base64.URLEncoding.DecodeString(att.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode attachment data: %w", err)
	}

	return data, nil
}

// DownloadInlineAttachment downloads an attachment that has inline data
func (c *Client) DownloadInlineAttachment(messageID string, partID string) ([]byte, error) {
	msg, err := c.Service.Users.Messages.Get(c.UserID, messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	part := findPart(msg.Payload, partID)
	if part == nil {
		return nil, fmt.Errorf("attachment part not found: %s", partID)
	}

	if part.Body == nil || part.Body.Data == "" {
		return nil, fmt.Errorf("attachment has no inline data")
	}

	data, err := base64.URLEncoding.DecodeString(part.Body.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode inline attachment: %w", err)
	}

	return data, nil
}

// findPart recursively finds a message part by its path (e.g., "0.1.2")
func findPart(payload *gmail.MessagePart, partPath string) *gmail.MessagePart {
	if partPath == "" {
		return payload
	}

	parts := strings.Split(partPath, ".")
	current := payload

	for _, indexStr := range parts {
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 0 || index >= len(current.Parts) {
			return nil
		}
		current = current.Parts[index]
	}

	return current
}

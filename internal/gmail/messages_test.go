package gmail

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestParseMessage(t *testing.T) {
	t.Run("extracts headers correctly", func(t *testing.T) {
		msg := &gmail.Message{
			Id:       "msg123",
			ThreadId: "thread456",
			Snippet:  "This is a test...",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "Test Subject"},
					{Name: "From", Value: "alice@example.com"},
					{Name: "To", Value: "bob@example.com"},
					{Name: "Date", Value: "Mon, 1 Jan 2024 12:00:00 +0000"},
				},
			},
		}

		result := parseMessage(msg, false, nil)

		assert.Equal(t, "msg123", result.ID)
		assert.Equal(t, "thread456", result.ThreadID)
		assert.Equal(t, "Test Subject", result.Subject)
		assert.Equal(t, "alice@example.com", result.From)
		assert.Equal(t, "bob@example.com", result.To)
		assert.Equal(t, "Mon, 1 Jan 2024 12:00:00 +0000", result.Date)
		assert.Equal(t, "This is a test...", result.Snippet)
	})

	t.Run("extracts thread ID", func(t *testing.T) {
		msg := &gmail.Message{
			Id:       "msg123",
			ThreadId: "thread789",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{},
			},
		}

		result := parseMessage(msg, false, nil)

		assert.Equal(t, "msg123", result.ID)
		assert.Equal(t, "thread789", result.ThreadID)
	})

	t.Run("handles nil payload", func(t *testing.T) {
		msg := &gmail.Message{
			Id:       "msg123",
			ThreadId: "thread456",
			Snippet:  "Preview text",
			Payload:  nil,
		}

		result := parseMessage(msg, true, nil)

		// Should not panic, basic fields populated
		assert.Equal(t, "msg123", result.ID)
		assert.Equal(t, "thread456", result.ThreadID)
		assert.Equal(t, "Preview text", result.Snippet)
		// Headers won't be extracted
		assert.Empty(t, result.Subject)
		assert.Empty(t, result.Body)
	})

	t.Run("handles case-insensitive headers", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{
					{Name: "SUBJECT", Value: "Upper Case"},
					{Name: "from", Value: "lower@example.com"},
					{Name: "To", Value: "mixed@example.com"},
				},
			},
		}

		result := parseMessage(msg, false, nil)

		assert.Equal(t, "Upper Case", result.Subject)
		assert.Equal(t, "lower@example.com", result.From)
		assert.Equal(t, "mixed@example.com", result.To)
	})

	t.Run("handles missing headers gracefully", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{},
			},
		}

		result := parseMessage(msg, false, nil)

		assert.Equal(t, "msg123", result.ID)
		assert.Empty(t, result.Subject)
		assert.Empty(t, result.From)
		assert.Empty(t, result.To)
		assert.Empty(t, result.Date)
	})
}

func TestExtractBody(t *testing.T) {
	t.Run("extracts plain text body", func(t *testing.T) {
		bodyText := "Hello, this is the message body."
		encoded := base64.URLEncoding.EncodeToString([]byte(bodyText))

		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body: &gmail.MessagePartBody{
				Data: encoded,
			},
		}

		result := extractBody(payload)
		assert.Equal(t, bodyText, result)
	})

	t.Run("extracts plain text from multipart message", func(t *testing.T) {
		bodyText := "Plain text content"
		encoded := base64.URLEncoding.EncodeToString([]byte(bodyText))

		payload := &gmail.MessagePart{
			MimeType: "multipart/alternative",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "text/plain",
					Body: &gmail.MessagePartBody{
						Data: encoded,
					},
				},
				{
					MimeType: "text/html",
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte("<p>HTML content</p>")),
					},
				},
			},
		}

		result := extractBody(payload)
		assert.Equal(t, bodyText, result)
	})

	t.Run("falls back to HTML if no plain text", func(t *testing.T) {
		htmlContent := "<p>HTML only</p>"
		encoded := base64.URLEncoding.EncodeToString([]byte(htmlContent))

		payload := &gmail.MessagePart{
			MimeType: "text/html",
			Body: &gmail.MessagePartBody{
				Data: encoded,
			},
		}

		result := extractBody(payload)
		assert.Equal(t, htmlContent, result)
	})

	t.Run("handles nested multipart", func(t *testing.T) {
		bodyText := "Nested plain text"
		encoded := base64.URLEncoding.EncodeToString([]byte(bodyText))

		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body: &gmail.MessagePartBody{
								Data: encoded,
							},
						},
					},
				},
			},
		}

		result := extractBody(payload)
		assert.Equal(t, bodyText, result)
	})

	t.Run("returns empty string for empty body", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{},
		}

		result := extractBody(payload)
		assert.Empty(t, result)
	})

	t.Run("returns empty string for nil body", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
		}

		result := extractBody(payload)
		assert.Empty(t, result)
	})

	t.Run("handles invalid base64 gracefully", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body: &gmail.MessagePartBody{
				Data: "not-valid-base64!!!",
			},
		}

		result := extractBody(payload)
		assert.Empty(t, result)
	})
}

func TestMessageStruct(t *testing.T) {
	t.Run("message struct has all fields", func(t *testing.T) {
		msg := &Message{
			ID:       "test-id",
			ThreadID: "thread-id",
			Subject:  "Test Subject",
			From:     "from@example.com",
			To:       "to@example.com",
			Date:     "2024-01-01",
			Snippet:  "Preview...",
			Body:     "Full body content",
		}

		assert.Equal(t, "test-id", msg.ID)
		assert.Equal(t, "thread-id", msg.ThreadID)
		assert.Equal(t, "Test Subject", msg.Subject)
		assert.Equal(t, "from@example.com", msg.From)
		assert.Equal(t, "to@example.com", msg.To)
		assert.Equal(t, "2024-01-01", msg.Date)
		assert.Equal(t, "Preview...", msg.Snippet)
		assert.Equal(t, "Full body content", msg.Body)
	})
}

func TestParseMessageWithBody(t *testing.T) {
	t.Run("includes body when requested", func(t *testing.T) {
		bodyText := "This is the full body"
		encoded := base64.URLEncoding.EncodeToString([]byte(bodyText))

		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				MimeType: "text/plain",
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "Test"},
				},
				Body: &gmail.MessagePartBody{
					Data: encoded,
				},
			},
		}

		result := parseMessage(msg, true, nil)
		assert.Equal(t, bodyText, result.Body)
	})

	t.Run("excludes body when not requested", func(t *testing.T) {
		bodyText := "This should not appear"
		encoded := base64.URLEncoding.EncodeToString([]byte(bodyText))

		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				MimeType: "text/plain",
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "Test"},
				},
				Body: &gmail.MessagePartBody{
					Data: encoded,
				},
			},
		}

		result := parseMessage(msg, false, nil)
		assert.Empty(t, result.Body)
	})
}

func TestExtractAttachments(t *testing.T) {
	t.Run("detects attachment by filename", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "text/plain",
					Body:     &gmail.MessagePartBody{Data: "body"},
				},
				{
					Filename: "report.pdf",
					MimeType: "application/pdf",
					Body: &gmail.MessagePartBody{
						Size:         12345,
						AttachmentId: "att123",
					},
				},
			},
		}

		attachments := extractAttachments(payload, "")
		assert.Len(t, attachments, 1)
		assert.Equal(t, "report.pdf", attachments[0].Filename)
		assert.Equal(t, "application/pdf", attachments[0].MimeType)
		assert.Equal(t, int64(12345), attachments[0].Size)
		assert.Equal(t, "att123", attachments[0].AttachmentID)
		assert.Equal(t, "1", attachments[0].PartID)
	})

	t.Run("detects attachment by Content-Disposition header", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					Filename: "data.csv",
					MimeType: "text/csv",
					Headers: []*gmail.MessagePartHeader{
						{Name: "Content-Disposition", Value: "attachment; filename=\"data.csv\""},
					},
					Body: &gmail.MessagePartBody{Size: 100},
				},
			},
		}

		attachments := extractAttachments(payload, "")
		assert.Len(t, attachments, 1)
		assert.Equal(t, "data.csv", attachments[0].Filename)
		assert.False(t, attachments[0].IsInline)
	})

	t.Run("detects inline attachment", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/related",
			Parts: []*gmail.MessagePart{
				{
					Filename: "image.png",
					MimeType: "image/png",
					Headers: []*gmail.MessagePartHeader{
						{Name: "Content-Disposition", Value: "inline; filename=\"image.png\""},
					},
					Body: &gmail.MessagePartBody{Size: 5000},
				},
			},
		}

		attachments := extractAttachments(payload, "")
		assert.Len(t, attachments, 1)
		assert.Equal(t, "image.png", attachments[0].Filename)
		assert.True(t, attachments[0].IsInline)
	})

	t.Run("handles nested multipart with multiple attachments", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{MimeType: "text/plain", Body: &gmail.MessagePartBody{Data: "text"}},
						{MimeType: "text/html", Body: &gmail.MessagePartBody{Data: "html"}},
					},
				},
				{
					Filename: "doc1.pdf",
					MimeType: "application/pdf",
					Body:     &gmail.MessagePartBody{Size: 1000, AttachmentId: "att1"},
				},
				{
					Filename: "doc2.pdf",
					MimeType: "application/pdf",
					Body:     &gmail.MessagePartBody{Size: 2000, AttachmentId: "att2"},
				},
			},
		}

		attachments := extractAttachments(payload, "")
		assert.Len(t, attachments, 2)
		assert.Equal(t, "doc1.pdf", attachments[0].Filename)
		assert.Equal(t, "1", attachments[0].PartID)
		assert.Equal(t, "doc2.pdf", attachments[1].Filename)
		assert.Equal(t, "2", attachments[1].PartID)
	})

	t.Run("handles message with no attachments", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{Data: "simple message"},
		}

		attachments := extractAttachments(payload, "")
		assert.Empty(t, attachments)
	})

	t.Run("generates correct part paths for deeply nested", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "multipart/related",
					Parts: []*gmail.MessagePart{
						{
							MimeType: "multipart/alternative",
							Parts: []*gmail.MessagePart{
								{MimeType: "text/plain", Body: &gmail.MessagePartBody{}},
							},
						},
						{
							Filename: "nested.png",
							MimeType: "image/png",
							Body:     &gmail.MessagePartBody{Size: 500},
						},
					},
				},
			},
		}

		attachments := extractAttachments(payload, "")
		assert.Len(t, attachments, 1)
		assert.Equal(t, "nested.png", attachments[0].Filename)
		assert.Equal(t, "0.1", attachments[0].PartID)
	})
}

func TestIsAttachment(t *testing.T) {
	t.Run("returns true for part with filename", func(t *testing.T) {
		part := &gmail.MessagePart{Filename: "test.pdf"}
		assert.True(t, isAttachment(part))
	})

	t.Run("returns true for Content-Disposition attachment", func(t *testing.T) {
		part := &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Content-Disposition", Value: "attachment; filename=\"test.pdf\""},
			},
		}
		assert.True(t, isAttachment(part))
	})

	t.Run("returns false for plain text part", func(t *testing.T) {
		part := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{Data: "text"},
		}
		assert.False(t, isAttachment(part))
	})

	t.Run("handles case-insensitive Content-Disposition", func(t *testing.T) {
		part := &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "CONTENT-DISPOSITION", Value: "ATTACHMENT"},
			},
		}
		assert.True(t, isAttachment(part))
	})
}

func TestIsInlineAttachment(t *testing.T) {
	t.Run("returns true for inline disposition", func(t *testing.T) {
		part := &gmail.MessagePart{
			Filename: "image.png",
			Headers: []*gmail.MessagePartHeader{
				{Name: "Content-Disposition", Value: "inline; filename=\"image.png\""},
			},
		}
		assert.True(t, isInlineAttachment(part))
	})

	t.Run("returns false for attachment disposition", func(t *testing.T) {
		part := &gmail.MessagePart{
			Filename: "doc.pdf",
			Headers: []*gmail.MessagePartHeader{
				{Name: "Content-Disposition", Value: "attachment; filename=\"doc.pdf\""},
			},
		}
		assert.False(t, isInlineAttachment(part))
	})

	t.Run("returns false for no disposition header", func(t *testing.T) {
		part := &gmail.MessagePart{Filename: "file.txt"}
		assert.False(t, isInlineAttachment(part))
	})
}

func TestParseMessageWithAttachments(t *testing.T) {
	t.Run("extracts attachments when body is requested", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				MimeType: "multipart/mixed",
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "With Attachment"},
				},
				Parts: []*gmail.MessagePart{
					{
						MimeType: "text/plain",
						Body: &gmail.MessagePartBody{
							Data: base64.URLEncoding.EncodeToString([]byte("body text")),
						},
					},
					{
						Filename: "attachment.pdf",
						MimeType: "application/pdf",
						Body:     &gmail.MessagePartBody{Size: 1234, AttachmentId: "att123"},
					},
				},
			},
		}

		result := parseMessage(msg, true, nil)
		assert.Equal(t, "body text", result.Body)
		assert.Len(t, result.Attachments, 1)
		assert.Equal(t, "attachment.pdf", result.Attachments[0].Filename)
	})

	t.Run("does not extract attachments when body not requested", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						Filename: "attachment.pdf",
						MimeType: "application/pdf",
						Body:     &gmail.MessagePartBody{Size: 1234},
					},
				},
			},
		}

		result := parseMessage(msg, false, nil)
		assert.Empty(t, result.Attachments)
	})
}

func TestExtractLabelsAndCategories(t *testing.T) {
	t.Run("separates user labels from categories", func(t *testing.T) {
		labelIds := []string{"Label_1", "CATEGORY_UPDATES", "Label_2", "CATEGORY_SOCIAL"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIds, resolver)

		assert.ElementsMatch(t, []string{"Label_1", "Label_2"}, labels)
		assert.ElementsMatch(t, []string{"updates", "social"}, categories)
	})

	t.Run("filters out system labels", func(t *testing.T) {
		labelIds := []string{"INBOX", "Label_1", "UNREAD", "STARRED", "IMPORTANT"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIds, resolver)

		assert.Equal(t, []string{"Label_1"}, labels)
		assert.Empty(t, categories)
	})

	t.Run("filters out CATEGORY_PERSONAL", func(t *testing.T) {
		labelIds := []string{"CATEGORY_PERSONAL", "CATEGORY_UPDATES"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIds, resolver)

		assert.Empty(t, labels)
		assert.Equal(t, []string{"updates"}, categories)
	})

	t.Run("uses resolver to translate label IDs", func(t *testing.T) {
		labelIds := []string{"Label_123", "Label_456"}
		resolver := func(id string) string {
			if id == "Label_123" {
				return "Work"
			}
			if id == "Label_456" {
				return "Personal"
			}
			return id
		}

		labels, categories := extractLabelsAndCategories(labelIds, resolver)

		assert.ElementsMatch(t, []string{"Work", "Personal"}, labels)
		assert.Empty(t, categories)
	})

	t.Run("handles nil resolver", func(t *testing.T) {
		labelIds := []string{"Label_1", "CATEGORY_SOCIAL"}

		labels, categories := extractLabelsAndCategories(labelIds, nil)

		assert.Equal(t, []string{"Label_1"}, labels)
		assert.Equal(t, []string{"social"}, categories)
	})

	t.Run("handles empty label IDs", func(t *testing.T) {
		labels, categories := extractLabelsAndCategories([]string{}, nil)

		assert.Empty(t, labels)
		assert.Empty(t, categories)
	})

	t.Run("handles nil label IDs", func(t *testing.T) {
		labels, categories := extractLabelsAndCategories(nil, nil)

		assert.Empty(t, labels)
		assert.Empty(t, categories)
	})
}

func TestParseMessageWithLabels(t *testing.T) {
	t.Run("extracts labels and categories from message", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{
					{Name: "Subject", Value: "Test"},
				},
			},
			LabelIds: []string{"Label_Work", "INBOX", "CATEGORY_UPDATES", "UNREAD"},
		}
		resolver := func(id string) string {
			if id == "Label_Work" {
				return "Work"
			}
			return id
		}

		result := parseMessage(msg, false, resolver)

		assert.Equal(t, []string{"Work"}, result.Labels)
		assert.Equal(t, []string{"updates"}, result.Categories)
	})

	t.Run("handles message with no labels", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{},
			},
			LabelIds: []string{},
		}

		result := parseMessage(msg, false, nil)

		assert.Empty(t, result.Labels)
		assert.Empty(t, result.Categories)
	})
}

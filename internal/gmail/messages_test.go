package gmail

import (
	"encoding/base64"
	"sort"
	"testing"

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

		if result.ID != "msg123" {
			t.Errorf("got %v, want %v", result.ID, "msg123")
		}
		if result.ThreadID != "thread456" {
			t.Errorf("got %v, want %v", result.ThreadID, "thread456")
		}
		if result.Subject != "Test Subject" {
			t.Errorf("got %v, want %v", result.Subject, "Test Subject")
		}
		if result.From != "alice@example.com" {
			t.Errorf("got %v, want %v", result.From, "alice@example.com")
		}
		if result.To != "bob@example.com" {
			t.Errorf("got %v, want %v", result.To, "bob@example.com")
		}
		if result.Date != "Mon, 1 Jan 2024 12:00:00 +0000" {
			t.Errorf("got %v, want %v", result.Date, "Mon, 1 Jan 2024 12:00:00 +0000")
		}
		if result.Snippet != "This is a test..." {
			t.Errorf("got %v, want %v", result.Snippet, "This is a test...")
		}
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

		if result.ID != "msg123" {
			t.Errorf("got %v, want %v", result.ID, "msg123")
		}
		if result.ThreadID != "thread789" {
			t.Errorf("got %v, want %v", result.ThreadID, "thread789")
		}
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
		if result.ID != "msg123" {
			t.Errorf("got %v, want %v", result.ID, "msg123")
		}
		if result.ThreadID != "thread456" {
			t.Errorf("got %v, want %v", result.ThreadID, "thread456")
		}
		if result.Snippet != "Preview text" {
			t.Errorf("got %v, want %v", result.Snippet, "Preview text")
		}
		// Headers won't be extracted
		if result.Subject != "" {
			t.Errorf("got %q, want empty", result.Subject)
		}
		if result.Body != "" {
			t.Errorf("got %q, want empty", result.Body)
		}
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

		if result.Subject != "Upper Case" {
			t.Errorf("got %v, want %v", result.Subject, "Upper Case")
		}
		if result.From != "lower@example.com" {
			t.Errorf("got %v, want %v", result.From, "lower@example.com")
		}
		if result.To != "mixed@example.com" {
			t.Errorf("got %v, want %v", result.To, "mixed@example.com")
		}
	})

	t.Run("handles missing headers gracefully", func(t *testing.T) {
		msg := &gmail.Message{
			Id: "msg123",
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{},
			},
		}

		result := parseMessage(msg, false, nil)

		if result.ID != "msg123" {
			t.Errorf("got %v, want %v", result.ID, "msg123")
		}
		if result.Subject != "" {
			t.Errorf("got %q, want empty", result.Subject)
		}
		if result.From != "" {
			t.Errorf("got %q, want empty", result.From)
		}
		if result.To != "" {
			t.Errorf("got %q, want empty", result.To)
		}
		if result.Date != "" {
			t.Errorf("got %q, want empty", result.Date)
		}
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
		if result != bodyText {
			t.Errorf("got %v, want %v", result, bodyText)
		}
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
		if result != bodyText {
			t.Errorf("got %v, want %v", result, bodyText)
		}
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
		if result != htmlContent {
			t.Errorf("got %v, want %v", result, htmlContent)
		}
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
		if result != bodyText {
			t.Errorf("got %v, want %v", result, bodyText)
		}
	})

	t.Run("returns empty string for empty body", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{},
		}

		result := extractBody(payload)
		if result != "" {
			t.Errorf("got %q, want empty", result)
		}
	})

	t.Run("returns empty string for nil body", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
		}

		result := extractBody(payload)
		if result != "" {
			t.Errorf("got %q, want empty", result)
		}
	})

	t.Run("handles invalid base64 gracefully", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body: &gmail.MessagePartBody{
				Data: "not-valid-base64!!!",
			},
		}

		result := extractBody(payload)
		if result != "" {
			t.Errorf("got %q, want empty", result)
		}
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

		if msg.ID != "test-id" {
			t.Errorf("got %v, want %v", msg.ID, "test-id")
		}
		if msg.ThreadID != "thread-id" {
			t.Errorf("got %v, want %v", msg.ThreadID, "thread-id")
		}
		if msg.Subject != "Test Subject" {
			t.Errorf("got %v, want %v", msg.Subject, "Test Subject")
		}
		if msg.From != "from@example.com" {
			t.Errorf("got %v, want %v", msg.From, "from@example.com")
		}
		if msg.To != "to@example.com" {
			t.Errorf("got %v, want %v", msg.To, "to@example.com")
		}
		if msg.Date != "2024-01-01" {
			t.Errorf("got %v, want %v", msg.Date, "2024-01-01")
		}
		if msg.Snippet != "Preview..." {
			t.Errorf("got %v, want %v", msg.Snippet, "Preview...")
		}
		if msg.Body != "Full body content" {
			t.Errorf("got %v, want %v", msg.Body, "Full body content")
		}
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
		if result.Body != bodyText {
			t.Errorf("got %v, want %v", result.Body, bodyText)
		}
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
		if result.Body != "" {
			t.Errorf("got %q, want empty", result.Body)
		}
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
		if len(attachments) != 1 {
			t.Errorf("got length %d, want %d", len(attachments), 1)
		}
		if attachments[0].Filename != "report.pdf" {
			t.Errorf("got %v, want %v", attachments[0].Filename, "report.pdf")
		}
		if attachments[0].MimeType != "application/pdf" {
			t.Errorf("got %v, want %v", attachments[0].MimeType, "application/pdf")
		}
		if attachments[0].Size != int64(12345) {
			t.Errorf("got %v, want %v", attachments[0].Size, int64(12345))
		}
		if attachments[0].AttachmentID != "att123" {
			t.Errorf("got %v, want %v", attachments[0].AttachmentID, "att123")
		}
		if attachments[0].PartID != "1" {
			t.Errorf("got %v, want %v", attachments[0].PartID, "1")
		}
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
		if len(attachments) != 1 {
			t.Errorf("got length %d, want %d", len(attachments), 1)
		}
		if attachments[0].Filename != "data.csv" {
			t.Errorf("got %v, want %v", attachments[0].Filename, "data.csv")
		}
		if attachments[0].IsInline {
			t.Error("got true, want false")
		}
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
		if len(attachments) != 1 {
			t.Errorf("got length %d, want %d", len(attachments), 1)
		}
		if attachments[0].Filename != "image.png" {
			t.Errorf("got %v, want %v", attachments[0].Filename, "image.png")
		}
		if !attachments[0].IsInline {
			t.Error("got false, want true")
		}
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
		if len(attachments) != 2 {
			t.Errorf("got length %d, want %d", len(attachments), 2)
		}
		if attachments[0].Filename != "doc1.pdf" {
			t.Errorf("got %v, want %v", attachments[0].Filename, "doc1.pdf")
		}
		if attachments[0].PartID != "1" {
			t.Errorf("got %v, want %v", attachments[0].PartID, "1")
		}
		if attachments[1].Filename != "doc2.pdf" {
			t.Errorf("got %v, want %v", attachments[1].Filename, "doc2.pdf")
		}
		if attachments[1].PartID != "2" {
			t.Errorf("got %v, want %v", attachments[1].PartID, "2")
		}
	})

	t.Run("handles message with no attachments", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{Data: "simple message"},
		}

		attachments := extractAttachments(payload, "")
		if len(attachments) != 0 {
			t.Errorf("got length %d, want 0", len(attachments))
		}
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
		if len(attachments) != 1 {
			t.Errorf("got length %d, want %d", len(attachments), 1)
		}
		if attachments[0].Filename != "nested.png" {
			t.Errorf("got %v, want %v", attachments[0].Filename, "nested.png")
		}
		if attachments[0].PartID != "0.1" {
			t.Errorf("got %v, want %v", attachments[0].PartID, "0.1")
		}
	})
}

func TestIsAttachment(t *testing.T) {
	t.Run("returns true for part with filename", func(t *testing.T) {
		part := &gmail.MessagePart{Filename: "test.pdf"}
		if !isAttachment(part) {
			t.Error("got false, want true")
		}
	})

	t.Run("returns true for Content-Disposition attachment", func(t *testing.T) {
		part := &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Content-Disposition", Value: "attachment; filename=\"test.pdf\""},
			},
		}
		if !isAttachment(part) {
			t.Error("got false, want true")
		}
	})

	t.Run("returns false for plain text part", func(t *testing.T) {
		part := &gmail.MessagePart{
			MimeType: "text/plain",
			Body:     &gmail.MessagePartBody{Data: "text"},
		}
		if isAttachment(part) {
			t.Error("got true, want false")
		}
	})

	t.Run("handles case-insensitive Content-Disposition", func(t *testing.T) {
		part := &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "CONTENT-DISPOSITION", Value: "ATTACHMENT"},
			},
		}
		if !isAttachment(part) {
			t.Error("got false, want true")
		}
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
		if !isInlineAttachment(part) {
			t.Error("got false, want true")
		}
	})

	t.Run("returns false for attachment disposition", func(t *testing.T) {
		part := &gmail.MessagePart{
			Filename: "doc.pdf",
			Headers: []*gmail.MessagePartHeader{
				{Name: "Content-Disposition", Value: "attachment; filename=\"doc.pdf\""},
			},
		}
		if isInlineAttachment(part) {
			t.Error("got true, want false")
		}
	})

	t.Run("returns false for no disposition header", func(t *testing.T) {
		part := &gmail.MessagePart{Filename: "file.txt"}
		if isInlineAttachment(part) {
			t.Error("got true, want false")
		}
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
		if result.Body != "body text" {
			t.Errorf("got %v, want %v", result.Body, "body text")
		}
		if len(result.Attachments) != 1 {
			t.Errorf("got length %d, want %d", len(result.Attachments), 1)
		}
		if result.Attachments[0].Filename != "attachment.pdf" {
			t.Errorf("got %v, want %v", result.Attachments[0].Filename, "attachment.pdf")
		}
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
		if len(result.Attachments) != 0 {
			t.Errorf("got length %d, want 0", len(result.Attachments))
		}
	})
}

func TestExtractLabelsAndCategories(t *testing.T) {
	t.Run("separates user labels from categories", func(t *testing.T) {
		labelIDs := []string{"Label_1", "CATEGORY_UPDATES", "Label_2", "CATEGORY_SOCIAL"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIDs, resolver)

		sort.Strings(labels)
		sort.Strings(categories)
		expectedLabels := []string{"Label_1", "Label_2"}
		expectedCategories := []string{"social", "updates"}
		sort.Strings(expectedLabels)
		sort.Strings(expectedCategories)

		if len(labels) != len(expectedLabels) {
			t.Fatalf("got labels length %d, want %d", len(labels), len(expectedLabels))
		}
		for i := range labels {
			if labels[i] != expectedLabels[i] {
				t.Errorf("labels[%d]: got %v, want %v", i, labels[i], expectedLabels[i])
			}
		}

		if len(categories) != len(expectedCategories) {
			t.Fatalf("got categories length %d, want %d", len(categories), len(expectedCategories))
		}
		for i := range categories {
			if categories[i] != expectedCategories[i] {
				t.Errorf("categories[%d]: got %v, want %v", i, categories[i], expectedCategories[i])
			}
		}
	})

	t.Run("filters out system labels", func(t *testing.T) {
		labelIDs := []string{"INBOX", "Label_1", "UNREAD", "STARRED", "IMPORTANT"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIDs, resolver)

		if len(labels) != 1 || labels[0] != "Label_1" {
			t.Errorf("got labels %v, want %v", labels, []string{"Label_1"})
		}
		if len(categories) != 0 {
			t.Errorf("got categories length %d, want 0", len(categories))
		}
	})

	t.Run("filters out CATEGORY_PERSONAL", func(t *testing.T) {
		labelIDs := []string{"CATEGORY_PERSONAL", "CATEGORY_UPDATES"}
		resolver := func(id string) string { return id }

		labels, categories := extractLabelsAndCategories(labelIDs, resolver)

		if len(labels) != 0 {
			t.Errorf("got labels length %d, want 0", len(labels))
		}
		if len(categories) != 1 || categories[0] != "updates" {
			t.Errorf("got categories %v, want %v", categories, []string{"updates"})
		}
	})

	t.Run("uses resolver to translate label IDs", func(t *testing.T) {
		labelIDs := []string{"Label_123", "Label_456"}
		resolver := func(id string) string {
			if id == "Label_123" {
				return "Work"
			}
			if id == "Label_456" {
				return "Personal"
			}
			return id
		}

		labels, categories := extractLabelsAndCategories(labelIDs, resolver)

		sort.Strings(labels)
		expectedLabels := []string{"Personal", "Work"}
		sort.Strings(expectedLabels)

		if len(labels) != len(expectedLabels) {
			t.Fatalf("got labels length %d, want %d", len(labels), len(expectedLabels))
		}
		for i := range labels {
			if labels[i] != expectedLabels[i] {
				t.Errorf("labels[%d]: got %v, want %v", i, labels[i], expectedLabels[i])
			}
		}
		if len(categories) != 0 {
			t.Errorf("got categories length %d, want 0", len(categories))
		}
	})

	t.Run("handles nil resolver", func(t *testing.T) {
		labelIDs := []string{"Label_1", "CATEGORY_SOCIAL"}

		labels, categories := extractLabelsAndCategories(labelIDs, nil)

		if len(labels) != 1 || labels[0] != "Label_1" {
			t.Errorf("got labels %v, want %v", labels, []string{"Label_1"})
		}
		if len(categories) != 1 || categories[0] != "social" {
			t.Errorf("got categories %v, want %v", categories, []string{"social"})
		}
	})

	t.Run("handles empty label IDs", func(t *testing.T) {
		labels, categories := extractLabelsAndCategories([]string{}, nil)

		if len(labels) != 0 {
			t.Errorf("got labels length %d, want 0", len(labels))
		}
		if len(categories) != 0 {
			t.Errorf("got categories length %d, want 0", len(categories))
		}
	})

	t.Run("handles nil label IDs", func(t *testing.T) {
		labels, categories := extractLabelsAndCategories(nil, nil)

		if len(labels) != 0 {
			t.Errorf("got labels length %d, want 0", len(labels))
		}
		if len(categories) != 0 {
			t.Errorf("got categories length %d, want 0", len(categories))
		}
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

		if len(result.Labels) != 1 || result.Labels[0] != "Work" {
			t.Errorf("got labels %v, want %v", result.Labels, []string{"Work"})
		}
		if len(result.Categories) != 1 || result.Categories[0] != "updates" {
			t.Errorf("got categories %v, want %v", result.Categories, []string{"updates"})
		}
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

		if len(result.Labels) != 0 {
			t.Errorf("got labels length %d, want 0", len(result.Labels))
		}
		if len(result.Categories) != 0 {
			t.Errorf("got categories length %d, want 0", len(result.Categories))
		}
	})
}

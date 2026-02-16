package gmail

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestFindPart(t *testing.T) {
	t.Parallel()
	t.Run("returns payload for empty path", func(t *testing.T) {
		t.Parallel()
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
		}
		result := findPart(payload, "")
		if result != payload {
			t.Errorf("got %v, want %v", result, payload)
		}
	})

	t.Run("finds part at index 0", func(t *testing.T) {
		t.Parallel()
		child := &gmail.MessagePart{MimeType: "text/plain", Filename: "file.txt"}
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{child},
		}
		result := findPart(payload, "0")
		if result != child {
			t.Errorf("got %v, want %v", result, child)
		}
	})

	t.Run("finds nested part", func(t *testing.T) {
		t.Parallel()
		deepChild := &gmail.MessagePart{MimeType: "application/pdf", Filename: "nested.pdf"}
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{MimeType: "text/plain"},
						deepChild,
					},
				},
			},
		}
		result := findPart(payload, "0.1")
		if result != deepChild {
			t.Errorf("got %v, want %v", result, deepChild)
		}
	})

	t.Run("returns nil for invalid index", func(t *testing.T) {
		t.Parallel()
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "5")
		if result != nil {
			t.Errorf("got %v, want nil", result)
		}
	})

	t.Run("returns nil for negative index", func(t *testing.T) {
		t.Parallel()
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "-1")
		if result != nil {
			t.Errorf("got %v, want nil", result)
		}
	})

	t.Run("returns nil for non-numeric path", func(t *testing.T) {
		t.Parallel()
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "abc")
		if result != nil {
			t.Errorf("got %v, want nil", result)
		}
	})

	t.Run("returns nil for out of bounds nested path", func(t *testing.T) {
		t.Parallel()
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts: []*gmail.MessagePart{
				{
					MimeType: "multipart/alternative",
					Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
				},
			},
		}
		result := findPart(payload, "0.5")
		if result != nil {
			t.Errorf("got %v, want nil", result)
		}
	})

	t.Run("handles deeply nested path", func(t *testing.T) {
		t.Parallel()
		deepest := &gmail.MessagePart{Filename: "deep.txt"}
		payload := &gmail.MessagePart{
			Parts: []*gmail.MessagePart{
				{
					Parts: []*gmail.MessagePart{
						{
							Parts: []*gmail.MessagePart{
								deepest,
							},
						},
					},
				},
			},
		}
		result := findPart(payload, "0.0.0")
		if result != deepest {
			t.Errorf("got %v, want %v", result, deepest)
		}
	})
}

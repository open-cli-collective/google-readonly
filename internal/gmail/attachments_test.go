package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestFindPart(t *testing.T) {
	t.Run("returns payload for empty path", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "text/plain",
		}
		result := findPart(payload, "")
		assert.Equal(t, payload, result)
	})

	t.Run("finds part at index 0", func(t *testing.T) {
		child := &gmail.MessagePart{MimeType: "text/plain", Filename: "file.txt"}
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{child},
		}
		result := findPart(payload, "0")
		assert.Equal(t, child, result)
	})

	t.Run("finds nested part", func(t *testing.T) {
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
		assert.Equal(t, deepChild, result)
	})

	t.Run("returns nil for invalid index", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "5")
		assert.Nil(t, result)
	})

	t.Run("returns nil for negative index", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "-1")
		assert.Nil(t, result)
	})

	t.Run("returns nil for non-numeric path", func(t *testing.T) {
		payload := &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Parts:    []*gmail.MessagePart{{MimeType: "text/plain"}},
		}
		result := findPart(payload, "abc")
		assert.Nil(t, result)
	})

	t.Run("returns nil for out of bounds nested path", func(t *testing.T) {
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
		assert.Nil(t, result)
	})

	t.Run("handles deeply nested path", func(t *testing.T) {
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
		assert.Equal(t, deepest, result)
	})
}

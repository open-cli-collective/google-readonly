package drive

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-cli-collective/google-readonly/internal/drive"
)

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "get <file-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"file-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"file-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "Get")
	})
}

func TestPrintFileDetails(t *testing.T) {
	// Capture stdout for testing
	captureOutput := func(fn func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		fn()

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("prints all fields for complete file", func(t *testing.T) {
		f := &drive.File{
			ID:           "abc123",
			Name:         "Test Document",
			MimeType:     drive.MimeTypeDocument,
			Size:         0, // Google Docs have no size
			CreatedTime:  time.Date(2024, 1, 10, 9, 30, 0, 0, time.UTC),
			ModifiedTime: time.Date(2024, 1, 15, 14, 22, 0, 0, time.UTC),
			Owners:       []string{"owner@example.com"},
			Shared:       true,
			WebViewLink:  "https://docs.google.com/document/d/abc123/edit",
			Parents:      []string{"parent123"},
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		assert.Contains(t, output, "File Details")
		assert.Contains(t, output, "ID:         abc123")
		assert.Contains(t, output, "Name:       Test Document")
		assert.Contains(t, output, "Type:       Document")
		assert.Contains(t, output, "Size:       -")
		assert.Contains(t, output, "Created:    2024-01-10 09:30:00")
		assert.Contains(t, output, "Modified:   2024-01-15 14:22:00")
		assert.Contains(t, output, "Owner:      owner@example.com")
		assert.Contains(t, output, "Shared:     Yes")
		assert.Contains(t, output, "Web Link:   https://docs.google.com/document/d/abc123/edit")
		assert.Contains(t, output, "Parent:     parent123")
	})

	t.Run("prints size for regular files", func(t *testing.T) {
		f := &drive.File{
			ID:       "xyz789",
			Name:     "photo.jpg",
			MimeType: "image/jpeg",
			Size:     1572864, // 1.5 MB
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		assert.Contains(t, output, "Size:       1.5 MB")
	})

	t.Run("handles unshared file", func(t *testing.T) {
		f := &drive.File{
			ID:     "private123",
			Name:   "private.txt",
			Shared: false,
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		assert.Contains(t, output, "Shared:     No")
	})

	t.Run("handles multiple owners", func(t *testing.T) {
		f := &drive.File{
			ID:     "shared123",
			Name:   "shared.txt",
			Owners: []string{"owner1@example.com", "owner2@example.com"},
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		assert.Contains(t, output, "Owner:      owner1@example.com, owner2@example.com")
	})

	t.Run("omits missing fields gracefully", func(t *testing.T) {
		f := &drive.File{
			ID:   "minimal123",
			Name: "minimal.txt",
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		assert.Contains(t, output, "ID:         minimal123")
		assert.Contains(t, output, "Name:       minimal.txt")
		// Should not contain empty values or crash
		assert.NotContains(t, output, "Created:")
		assert.NotContains(t, output, "Modified:")
		assert.NotContains(t, output, "Owner:")
		assert.NotContains(t, output, "Web Link:")
		assert.NotContains(t, output, "Parent:")
	})
}

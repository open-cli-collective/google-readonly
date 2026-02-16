package drive

import (
	"testing"
	"time"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestGetCommand(t *testing.T) {
	cmd := newGetCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "get <file-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"file-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"file-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "Get")
	})
}

func TestPrintFileDetails(t *testing.T) {
	captureOutput := func(fn func()) string {
		return testutil.CaptureStdout(t, fn)
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

		testutil.Contains(t, output, "File Details")
		testutil.Contains(t, output, "ID:         abc123")
		testutil.Contains(t, output, "Name:       Test Document")
		testutil.Contains(t, output, "Type:       Document")
		testutil.Contains(t, output, "Size:       -")
		testutil.Contains(t, output, "Created:    2024-01-10 09:30:00")
		testutil.Contains(t, output, "Modified:   2024-01-15 14:22:00")
		testutil.Contains(t, output, "Owner:      owner@example.com")
		testutil.Contains(t, output, "Shared:     Yes")
		testutil.Contains(t, output, "Web Link:   https://docs.google.com/document/d/abc123/edit")
		testutil.Contains(t, output, "Parent:     parent123")
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

		testutil.Contains(t, output, "Size:       1.5 MB")
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

		testutil.Contains(t, output, "Shared:     No")
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

		testutil.Contains(t, output, "Owner:      owner1@example.com, owner2@example.com")
	})

	t.Run("omits missing fields gracefully", func(t *testing.T) {
		f := &drive.File{
			ID:   "minimal123",
			Name: "minimal.txt",
		}

		output := captureOutput(func() {
			printFileDetails(f)
		})

		testutil.Contains(t, output, "ID:         minimal123")
		testutil.Contains(t, output, "Name:       minimal.txt")
		// Should not contain empty values or crash
		testutil.NotContains(t, output, "Created:")
		testutil.NotContains(t, output, "Modified:")
		testutil.NotContains(t, output, "Owner:")
		testutil.NotContains(t, output, "Web Link:")
		testutil.NotContains(t, output, "Parent:")
	})
}

package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsZipFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		mimeType string
		expected bool
	}{
		{"zip extension lowercase", "archive.zip", "", true},
		{"zip extension uppercase", "ARCHIVE.ZIP", "", true},
		{"zip extension mixed case", "Archive.Zip", "", true},
		{"application/zip mime type", "archive", "application/zip", true},
		{"application/x-zip-compressed mime type", "archive", "application/x-zip-compressed", true},
		{"pdf file", "document.pdf", "application/pdf", false},
		{"txt file", "readme.txt", "text/plain", false},
		{"no extension with wrong mime", "archive", "application/octet-stream", false},
		{"zip extension with different mime", "archive.zip", "application/octet-stream", true},
		{"empty filename with zip mime", "", "application/zip", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZipFile(tt.filename, tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 500, "500 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1MB", 1024 * 1024, "1.0 MB"},
		{"2.5 MB", 2621440, "2.5 MB"},
		{"exactly 1GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"large file", 5368709120, "5.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAttachmentsCommand(t *testing.T) {
	cmd := newAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "attachments", cmd.Use)
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 2)

		var names []string
		for _, cmd := range subcommands {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "download")
	})
}

func TestListAttachmentsCommand(t *testing.T) {
	cmd := newListAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "list <message-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		assert.NoError(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
	})
}

func TestDownloadAttachmentsCommand(t *testing.T) {
	cmd := newDownloadAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "download <message-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		assert.NoError(t, err)
	})

	t.Run("has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"filename", "f"},
			{"output", "o"},
			{"extract", "e"},
			{"all", "a"},
		}

		for _, f := range flags {
			flag := cmd.Flags().Lookup(f.name)
			assert.NotNil(t, flag, "flag %s should exist", f.name)
			assert.Equal(t, f.shorthand, flag.Shorthand, "flag %s should have shorthand %s", f.name, f.shorthand)
		}
	})
}

package mail

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
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
			testutil.Equal(t, result, tt.expected)
		})
	}
}

// Tests for formatSize moved to internal/format/format_test.go

func TestAttachmentsCommand(t *testing.T) {
	cmd := newAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "attachments")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		testutil.GreaterOrEqual(t, len(subcommands), 2)

		var names []string
		for _, cmd := range subcommands {
			names = append(names, cmd.Name())
		}
		testutil.SliceContains(t, names, "list")
		testutil.SliceContains(t, names, "download")
	})
}

func TestListAttachmentsCommand(t *testing.T) {
	cmd := newListAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "list <message-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		testutil.NoError(t, err)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
	})
}

func TestDownloadAttachmentsCommand(t *testing.T) {
	cmd := newDownloadAttachmentsCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "download <message-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"msg123"})
		testutil.NoError(t, err)
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
			testutil.NotNil(t, flag)
			testutil.Equal(t, flag.Shorthand, f.shorthand)
		}
	})
}

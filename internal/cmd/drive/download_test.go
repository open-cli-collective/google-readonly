package drive

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestDownloadCommand(t *testing.T) {
	cmd := newDownloadCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "download <file-id>")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.Error(t, err)

		err = cmd.Args(cmd, []string{"file-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"file-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has output flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("output")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "o")
	})

	t.Run("has format flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "f")
	})

	t.Run("has stdout flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("stdout")
		testutil.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "Download")
	})
}

func TestDetermineOutputPath(t *testing.T) {
	t.Run("uses user-specified output path", func(t *testing.T) {
		result := determineOutputPath("original.doc", "pdf", "/custom/path.pdf")
		testutil.Equal(t, result, "/custom/path.pdf")
	})

	t.Run("uses original name when no format or output", func(t *testing.T) {
		result := determineOutputPath("document.pdf", "", "")
		testutil.Equal(t, result, "document.pdf")
	})

	t.Run("replaces extension when format specified", func(t *testing.T) {
		result := determineOutputPath("Report", "pdf", "")
		testutil.Equal(t, result, "Report.pdf")
	})

	t.Run("replaces existing extension when format specified", func(t *testing.T) {
		result := determineOutputPath("Report.gdoc", "docx", "")
		testutil.Equal(t, result, "Report.docx")
	})

	t.Run("handles various export formats", func(t *testing.T) {
		tests := []struct {
			name     string
			format   string
			expected string
		}{
			{"Document", "pdf", "Document.pdf"},
			{"Document", "docx", "Document.docx"},
			{"Document", "txt", "Document.txt"},
			{"Spreadsheet", "xlsx", "Spreadsheet.xlsx"},
			{"Spreadsheet", "csv", "Spreadsheet.csv"},
			{"Presentation", "pptx", "Presentation.pptx"},
			{"Drawing", "png", "Drawing.png"},
			{"Drawing", "svg", "Drawing.svg"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				result := determineOutputPath(tt.name, tt.format, "")
				testutil.Equal(t, result, tt.expected)
			})
		}
	})
}

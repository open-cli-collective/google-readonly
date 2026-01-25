package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadCommand(t *testing.T) {
	cmd := newDownloadCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "download <file-id>", cmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"file-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"file-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has output flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("output")
		assert.NotNil(t, flag)
		assert.Equal(t, "o", flag.Shorthand)
	})

	t.Run("has format flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "f", flag.Shorthand)
	})

	t.Run("has stdout flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("stdout")
		assert.NotNil(t, flag)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "Download")
	})
}

func TestDetermineOutputPath(t *testing.T) {
	t.Run("uses user-specified output path", func(t *testing.T) {
		result := determineOutputPath("original.doc", "pdf", "/custom/path.pdf")
		assert.Equal(t, "/custom/path.pdf", result)
	})

	t.Run("uses original name when no format or output", func(t *testing.T) {
		result := determineOutputPath("document.pdf", "", "")
		assert.Equal(t, "document.pdf", result)
	})

	t.Run("replaces extension when format specified", func(t *testing.T) {
		result := determineOutputPath("Report", "pdf", "")
		assert.Equal(t, "Report.pdf", result)
	})

	t.Run("replaces existing extension when format specified", func(t *testing.T) {
		result := determineOutputPath("Report.gdoc", "docx", "")
		assert.Equal(t, "Report.docx", result)
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
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

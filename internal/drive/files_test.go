package drive

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
)

func TestParseFile(t *testing.T) {
	t.Run("parses basic file", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "test.txt",
			MimeType:     "text/plain",
			Size:         1024,
			CreatedTime:  "2024-01-15T10:30:00Z",
			ModifiedTime: "2024-01-16T14:00:00Z",
			Parents:      []string{"parent1"},
			WebViewLink:  "https://drive.google.com/file/d/123",
			Shared:       true,
		}

		result := ParseFile(f)

		if result.ID != "123" {
			t.Errorf("got %v, want %v", result.ID, "123")
		}
		if result.Name != "test.txt" {
			t.Errorf("got %v, want %v", result.Name, "test.txt")
		}
		if result.MimeType != "text/plain" {
			t.Errorf("got %v, want %v", result.MimeType, "text/plain")
		}
		if result.Size != int64(1024) {
			t.Errorf("got %v, want %v", result.Size, int64(1024))
		}
		if result.CreatedTime.Year() != 2024 {
			t.Errorf("got %v, want %v", result.CreatedTime.Year(), 2024)
		}
		if result.ModifiedTime.Year() != 2024 {
			t.Errorf("got %v, want %v", result.ModifiedTime.Year(), 2024)
		}
		if !reflect.DeepEqual(result.Parents, []string{"parent1"}) {
			t.Errorf("got %v, want %v", result.Parents, []string{"parent1"})
		}
		if result.WebViewLink != "https://drive.google.com/file/d/123" {
			t.Errorf("got %v, want %v", result.WebViewLink, "https://drive.google.com/file/d/123")
		}
		if !result.Shared {
			t.Error("got false, want true")
		}
	})

	t.Run("parses file with owners", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "shared.txt",
			MimeType: "text/plain",
			Owners: []*drive.User{
				{EmailAddress: "owner1@example.com"},
				{EmailAddress: "owner2@example.com"},
			},
		}

		result := ParseFile(f)

		expected := []string{"owner1@example.com", "owner2@example.com"}
		if !reflect.DeepEqual(result.Owners, expected) {
			t.Errorf("got %v, want %v", result.Owners, expected)
		}
	})

	t.Run("handles empty timestamps", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "no-times.txt",
			MimeType:     "text/plain",
			CreatedTime:  "",
			ModifiedTime: "",
		}

		result := ParseFile(f)

		if !result.CreatedTime.IsZero() {
			t.Error("got false, want true")
		}
		if !result.ModifiedTime.IsZero() {
			t.Error("got false, want true")
		}
	})

	t.Run("handles malformed timestamps", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "bad-times.txt",
			MimeType:     "text/plain",
			CreatedTime:  "not-a-timestamp",
			ModifiedTime: "also-not-valid",
		}

		result := ParseFile(f)

		if !result.CreatedTime.IsZero() {
			t.Error("got false, want true")
		}
		if !result.ModifiedTime.IsZero() {
			t.Error("got false, want true")
		}
	})

	t.Run("handles nil owners", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "no-owners.txt",
			MimeType: "text/plain",
			Owners:   nil,
		}

		result := ParseFile(f)

		if result.Owners != nil {
			t.Errorf("got %v, want nil", result.Owners)
		}
	})

	t.Run("handles empty owners slice", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "empty-owners.txt",
			MimeType: "text/plain",
			Owners:   []*drive.User{},
		}

		result := ParseFile(f)

		if result.Owners != nil {
			t.Errorf("got %v, want nil", result.Owners)
		}
	})
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		// Google Workspace types
		{MimeTypeFolder, "Folder"},
		{MimeTypeDocument, "Document"},
		{MimeTypeSpreadsheet, "Spreadsheet"},
		{MimeTypePresentation, "Presentation"},
		{MimeTypeDrawing, "Drawing"},
		{MimeTypeForm, "Form"},
		{MimeTypeSite, "Site"},
		{MimeTypeShortcut, "Shortcut"},
		// Common file types
		{"application/pdf", "PDF"},
		{"text/plain", "Text"},
		{"text/html", "HTML"},
		{"text/csv", "CSV"},
		{"application/zip", "ZIP"},
		// Prefix-based types
		{"image/png", "Image"},
		{"image/jpeg", "Image"},
		{"video/mp4", "Video"},
		{"video/quicktime", "Video"},
		{"audio/mpeg", "Audio"},
		{"audio/wav", "Audio"},
		// Unknown types
		{"application/octet-stream", "application/octet-stream"},
		{"unknown/type", "unknown/type"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := GetTypeName(tt.mimeType)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsGoogleWorkspaceFile(t *testing.T) {
	t.Run("returns true for Google Workspace files", func(t *testing.T) {
		workspaceTypes := []string{
			MimeTypeDocument,
			MimeTypeSpreadsheet,
			MimeTypePresentation,
			MimeTypeDrawing,
			MimeTypeForm,
			MimeTypeSite,
		}

		for _, mimeType := range workspaceTypes {
			if !IsGoogleWorkspaceFile(mimeType) {
				t.Errorf("got false, want true for %s", mimeType)
			}
		}
	})

	t.Run("returns false for non-Workspace files", func(t *testing.T) {
		nonWorkspaceTypes := []string{
			MimeTypeFolder,
			MimeTypeShortcut,
			"application/pdf",
			"text/plain",
			"image/png",
			"video/mp4",
			"application/octet-stream",
		}

		for _, mimeType := range nonWorkspaceTypes {
			if IsGoogleWorkspaceFile(mimeType) {
				t.Errorf("got true, want false for %s", mimeType)
			}
		}
	})
}

func TestGetExportMimeType(t *testing.T) {
	t.Run("returns correct MIME type for Document exports", func(t *testing.T) {
		tests := []struct {
			format   string
			expected string
		}{
			{"pdf", "application/pdf"},
			{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
			{"txt", "text/plain"},
			{"html", "text/html"},
			{"md", "text/markdown"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				result, err := GetExportMimeType(MimeTypeDocument, tt.format)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("returns correct MIME type for Spreadsheet exports", func(t *testing.T) {
		tests := []struct {
			format   string
			expected string
		}{
			{"pdf", "application/pdf"},
			{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
			{"csv", "text/csv"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				result, err := GetExportMimeType(MimeTypeSpreadsheet, tt.format)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("returns correct MIME type for Presentation exports", func(t *testing.T) {
		result, err := GetExportMimeType(MimeTypePresentation, "pptx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "application/vnd.openxmlformats-officedocument.presentationml.presentation" {
			t.Errorf("got %v, want %v", result, "application/vnd.openxmlformats-officedocument.presentationml.presentation")
		}
	})

	t.Run("returns correct MIME type for Drawing exports", func(t *testing.T) {
		result, err := GetExportMimeType(MimeTypeDrawing, "png")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "image/png" {
			t.Errorf("got %v, want %v", result, "image/png")
		}
	})

	t.Run("returns error for unsupported format", func(t *testing.T) {
		_, err := GetExportMimeType(MimeTypeDocument, "xyz")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("expected %q to contain %q", err.Error(), "not supported")
		}
	})

	t.Run("returns error for non-exportable file type", func(t *testing.T) {
		_, err := GetExportMimeType("application/pdf", "docx")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "does not support export") {
			t.Errorf("expected %q to contain %q", err.Error(), "does not support export")
		}
	})

	t.Run("returns error for format not matching file type", func(t *testing.T) {
		// csv is valid for spreadsheets but not documents
		_, err := GetExportMimeType(MimeTypeDocument, "csv")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not supported for Google Document") {
			t.Errorf("expected %q to contain %q", err.Error(), "not supported for Google Document")
		}
	})
}

func TestGetSupportedExportFormats(t *testing.T) {
	t.Run("returns formats for Document", func(t *testing.T) {
		formats := GetSupportedExportFormats(MimeTypeDocument)
		if !slices.Contains(formats, "pdf") {
			t.Errorf("expected formats to contain %q", "pdf")
		}
		if !slices.Contains(formats, "docx") {
			t.Errorf("expected formats to contain %q", "docx")
		}
		if !slices.Contains(formats, "txt") {
			t.Errorf("expected formats to contain %q", "txt")
		}
	})

	t.Run("returns formats for Spreadsheet", func(t *testing.T) {
		formats := GetSupportedExportFormats(MimeTypeSpreadsheet)
		if !slices.Contains(formats, "xlsx") {
			t.Errorf("expected formats to contain %q", "xlsx")
		}
		if !slices.Contains(formats, "csv") {
			t.Errorf("expected formats to contain %q", "csv")
		}
	})

	t.Run("returns nil for non-exportable file", func(t *testing.T) {
		formats := GetSupportedExportFormats("application/pdf")
		if formats != nil {
			t.Errorf("got %v, want nil", formats)
		}
	})
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"pdf", ".pdf"},
		{"docx", ".docx"},
		{"xlsx", ".xlsx"},
		{"pptx", ".pptx"},
		{"txt", ".txt"},
		{"html", ".html"},
		{"md", ".md"},
		{"csv", ".csv"},
		{"png", ".png"},
		{"svg", ".svg"},
		{"jpg", ".jpg"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := GetFileExtension(tt.format)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

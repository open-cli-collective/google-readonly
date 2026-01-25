package drive

import (
	"time"

	"google.golang.org/api/drive/v3"
)

// File represents a Google Drive file with simplified fields for JSON output
type File struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MimeType     string    `json:"mimeType"`
	Size         int64     `json:"size,omitempty"`
	CreatedTime  time.Time `json:"createdTime,omitempty"`
	ModifiedTime time.Time `json:"modifiedTime,omitempty"`
	Parents      []string  `json:"parents,omitempty"`
	Owners       []string  `json:"owners,omitempty"`
	WebViewLink  string    `json:"webViewLink,omitempty"`
	Shared       bool      `json:"shared"`
}

// ParseFile converts a Google Drive API File to our simplified File struct
func ParseFile(f *drive.File) *File {
	file := &File{
		ID:          f.Id,
		Name:        f.Name,
		MimeType:    f.MimeType,
		Size:        f.Size,
		Parents:     f.Parents,
		WebViewLink: f.WebViewLink,
		Shared:      f.Shared,
	}

	// Parse timestamps
	if f.CreatedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.CreatedTime); err == nil {
			file.CreatedTime = t
		}
	}
	if f.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
			file.ModifiedTime = t
		}
	}

	// Extract owner emails
	if len(f.Owners) > 0 {
		file.Owners = make([]string, 0, len(f.Owners))
		for _, owner := range f.Owners {
			file.Owners = append(file.Owners, owner.EmailAddress)
		}
	}

	return file
}

// MIME type constants for Google Workspace files
const (
	MimeTypeFolder       = "application/vnd.google-apps.folder"
	MimeTypeDocument     = "application/vnd.google-apps.document"
	MimeTypeSpreadsheet  = "application/vnd.google-apps.spreadsheet"
	MimeTypePresentation = "application/vnd.google-apps.presentation"
	MimeTypeDrawing      = "application/vnd.google-apps.drawing"
	MimeTypeForm         = "application/vnd.google-apps.form"
	MimeTypeSite         = "application/vnd.google-apps.site"
	MimeTypeShortcut     = "application/vnd.google-apps.shortcut"
)

// GetTypeName returns a human-readable name for a MIME type
func GetTypeName(mimeType string) string {
	switch mimeType {
	case MimeTypeFolder:
		return "Folder"
	case MimeTypeDocument:
		return "Document"
	case MimeTypeSpreadsheet:
		return "Spreadsheet"
	case MimeTypePresentation:
		return "Presentation"
	case MimeTypeDrawing:
		return "Drawing"
	case MimeTypeForm:
		return "Form"
	case MimeTypeSite:
		return "Site"
	case MimeTypeShortcut:
		return "Shortcut"
	case "application/pdf":
		return "PDF"
	case "text/plain":
		return "Text"
	case "text/html":
		return "HTML"
	case "text/csv":
		return "CSV"
	case "application/zip":
		return "ZIP"
	default:
		// Check for common prefixes
		if len(mimeType) > 6 {
			switch mimeType[:6] {
			case "image/":
				return "Image"
			case "video/":
				return "Video"
			case "audio/":
				return "Audio"
			}
		}
		return mimeType
	}
}

// IsGoogleWorkspaceFile returns true if the MIME type is a Google Workspace file
// that requires export rather than direct download
func IsGoogleWorkspaceFile(mimeType string) bool {
	switch mimeType {
	case MimeTypeDocument, MimeTypeSpreadsheet, MimeTypePresentation,
		MimeTypeDrawing, MimeTypeForm, MimeTypeSite:
		return true
	default:
		return false
	}
}

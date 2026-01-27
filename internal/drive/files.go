package drive

import (
	"fmt"
	"strings"
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
	DriveID      string    `json:"driveId,omitempty"` // Shared drive ID if file is in a shared drive
}

// SharedDrive represents a Google Shared Drive (formerly Team Drive)
type SharedDrive struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DriveScope defines where to search for files
type DriveScope struct {
	AllDrives   bool   // Search everywhere (My Drive + all shared drives)
	MyDriveOnly bool   // Restrict to personal My Drive only
	DriveID     string // Specific shared drive ID
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
		DriveID:     f.DriveId,
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

// Export format to MIME type mappings
var documentExportFormats = map[string]string{
	"pdf":  "application/pdf",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"txt":  "text/plain",
	"html": "text/html",
	"md":   "text/markdown",
	"rtf":  "application/rtf",
	"odt":  "application/vnd.oasis.opendocument.text",
}

var spreadsheetExportFormats = map[string]string{
	"pdf":  "application/pdf",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"csv":  "text/csv",
	"tsv":  "text/tab-separated-values",
	"ods":  "application/vnd.oasis.opendocument.spreadsheet",
}

var presentationExportFormats = map[string]string{
	"pdf":  "application/pdf",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"odp":  "application/vnd.oasis.opendocument.presentation",
}

var drawingExportFormats = map[string]string{
	"pdf": "application/pdf",
	"png": "image/png",
	"svg": "image/svg+xml",
	"jpg": "image/jpeg",
}

// GetExportMimeType returns the MIME type for exporting a Google Workspace file
// to the specified format. Returns an error if the format is not supported.
func GetExportMimeType(sourceMimeType, format string) (string, error) {
	var formats map[string]string
	var typeName string

	switch sourceMimeType {
	case MimeTypeDocument:
		formats = documentExportFormats
		typeName = "Document"
	case MimeTypeSpreadsheet:
		formats = spreadsheetExportFormats
		typeName = "Spreadsheet"
	case MimeTypePresentation:
		formats = presentationExportFormats
		typeName = "Presentation"
	case MimeTypeDrawing:
		formats = drawingExportFormats
		typeName = "Drawing"
	default:
		return "", fmt.Errorf("file type %s does not support export", GetTypeName(sourceMimeType))
	}

	mimeType, ok := formats[format]
	if !ok {
		return "", fmt.Errorf("format '%s' not supported for Google %s (supported: %s)",
			format, typeName, getSupportedFormats(formats))
	}
	return mimeType, nil
}

// GetSupportedExportFormats returns the supported export formats for a Google Workspace file type
func GetSupportedExportFormats(sourceMimeType string) []string {
	var formats map[string]string

	switch sourceMimeType {
	case MimeTypeDocument:
		formats = documentExportFormats
	case MimeTypeSpreadsheet:
		formats = spreadsheetExportFormats
	case MimeTypePresentation:
		formats = presentationExportFormats
	case MimeTypeDrawing:
		formats = drawingExportFormats
	default:
		return nil
	}

	result := make([]string, 0, len(formats))
	for format := range formats {
		result = append(result, format)
	}
	return result
}

func getSupportedFormats(formats map[string]string) string {
	result := make([]string, 0, len(formats))
	for format := range formats {
		result = append(result, format)
	}
	return strings.Join(result, ", ")
}

// GetFileExtension returns the appropriate file extension for a format
func GetFileExtension(format string) string {
	switch format {
	case "docx", "xlsx", "pptx", "pdf", "txt", "html", "rtf", "odt", "csv", "tsv", "ods", "odp", "png", "svg", "jpg":
		return "." + format
	case "md":
		return ".md"
	default:
		return ""
	}
}

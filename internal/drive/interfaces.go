package drive

// DriveClientInterface defines the interface for Drive client operations.
// This enables unit testing through mock implementations.
type DriveClientInterface interface {
	// ListFiles returns files matching the query
	ListFiles(query string, pageSize int64) ([]*File, error)

	// GetFile retrieves a single file by ID
	GetFile(fileID string) (*File, error)

	// DownloadFile downloads a regular (non-Google Workspace) file
	DownloadFile(fileID string) ([]byte, error)

	// ExportFile exports a Google Workspace file to the specified MIME type
	ExportFile(fileID string, mimeType string) ([]byte, error)
}

// Verify that Client implements DriveClientInterface
var _ DriveClientInterface = (*Client)(nil)

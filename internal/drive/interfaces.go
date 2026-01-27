package drive

// DriveClientInterface defines the interface for Drive client operations.
// This enables unit testing through mock implementations.
type DriveClientInterface interface {
	// ListFiles returns files matching the query (searches My Drive only for backwards compatibility)
	ListFiles(query string, pageSize int64) ([]*File, error)

	// ListFilesWithScope returns files matching the query within the specified scope
	ListFilesWithScope(query string, pageSize int64, scope DriveScope) ([]*File, error)

	// GetFile retrieves a single file by ID (supports all drives)
	GetFile(fileID string) (*File, error)

	// DownloadFile downloads a regular (non-Google Workspace) file
	DownloadFile(fileID string) ([]byte, error)

	// ExportFile exports a Google Workspace file to the specified MIME type
	ExportFile(fileID string, mimeType string) ([]byte, error)

	// ListSharedDrives returns all shared drives accessible to the user
	ListSharedDrives(pageSize int64) ([]*SharedDrive, error)
}

// Verify that Client implements DriveClientInterface
var _ DriveClientInterface = (*Client)(nil)

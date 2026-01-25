package drive

// DriveClientInterface defines the interface for Drive client operations.
// This enables unit testing through mock implementations.
type DriveClientInterface interface {
	// ListFiles returns files matching the query
	ListFiles(query string, pageSize int64) ([]*File, error)

	// GetFile retrieves a single file by ID
	GetFile(fileID string) (*File, error)
}

// Verify that Client implements DriveClientInterface
var _ DriveClientInterface = (*Client)(nil)

package drive

import (
	"context"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// DriveClient defines the interface for Drive client operations used by drive commands.
type DriveClient interface {
	ListFiles(query string, pageSize int64) ([]*drive.File, error)
	ListFilesWithScope(query string, pageSize int64, scope drive.DriveScope) ([]*drive.File, error)
	GetFile(fileID string) (*drive.File, error)
	DownloadFile(fileID string) ([]byte, error)
	ExportFile(fileID string, mimeType string) ([]byte, error)
	ListSharedDrives(pageSize int64) ([]*drive.SharedDrive, error)
}

// ClientFactory is the function used to create Drive clients.
// Override in tests to inject mocks.
var ClientFactory = func() (DriveClient, error) {
	return drive.NewClient(context.Background())
}

// newDriveClient creates and returns a new Drive client
func newDriveClient() (DriveClient, error) {
	return ClientFactory()
}

// printJSON encodes data as indented JSON to stdout
func printJSON(data any) error {
	return output.JSONStdout(data)
}

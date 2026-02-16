package drive

import (
	"context"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// DriveClient defines the interface for Drive client operations used by drive commands.
type DriveClient interface {
	ListFiles(ctx context.Context, query string, pageSize int64) ([]*drive.File, error)
	ListFilesWithScope(ctx context.Context, query string, pageSize int64, scope drive.DriveScope) ([]*drive.File, error)
	GetFile(ctx context.Context, fileID string) (*drive.File, error)
	DownloadFile(ctx context.Context, fileID string) ([]byte, error)
	ExportFile(ctx context.Context, fileID string, mimeType string) ([]byte, error)
	ListSharedDrives(ctx context.Context, pageSize int64) ([]*drive.SharedDrive, error)
}

// ClientFactory is the function used to create Drive clients.
// Override in tests to inject mocks.
var ClientFactory = func(ctx context.Context) (DriveClient, error) {
	return drive.NewClient(ctx)
}

// newDriveClient creates and returns a new Drive client
func newDriveClient(ctx context.Context) (DriveClient, error) {
	return ClientFactory(ctx)
}

// printJSON encodes data as indented JSON to stdout
func printJSON(data any) error {
	return output.JSONStdout(data)
}

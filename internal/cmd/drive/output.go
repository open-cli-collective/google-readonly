package drive

import (
	"context"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// ClientFactory is the function used to create Drive clients.
// Override in tests to inject mocks.
var ClientFactory = func() (drive.DriveClientInterface, error) {
	return drive.NewClient(context.Background())
}

// newDriveClient creates and returns a new Drive client
func newDriveClient() (drive.DriveClientInterface, error) {
	return ClientFactory()
}

// printJSON encodes data as indented JSON to stdout
func printJSON(data any) error {
	return output.JSONStdout(data)
}

package drive

import (
	"context"
	"encoding/json"
	"os"

	"github.com/open-cli-collective/google-readonly/internal/drive"
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
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

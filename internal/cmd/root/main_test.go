package root

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli-common/config"

	"github.com/open-cli-collective/google-readonly/internal/appidentity"
)

// TestMain registers gro's real identity before any test runs, mirroring what
// main does at startup, so config/keychain/auth paths and the scope set resolve.
func TestMain(m *testing.M) {
	config.Register(appidentity.Identity())
	os.Exit(m.Run())
}

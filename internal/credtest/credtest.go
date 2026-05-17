// Package credtest provides a hermetic credential environment for tests
// (§1.12 test obligation). It forces credstore's encrypted-file backend
// inside a per-test temp HOME with a fixed passphrase, so no test ever
// touches the real OS keyring, shells out to `security`/`secret-tool`, or
// depends on machine state.
//
// It is deliberately a tiny leaf (only testing + migrationsink) so that the
// white-box `package keychain` tests can import it without the
// keychain<->testutil import cycle that the fixture-heavy internal/testutil
// would introduce.
package credtest

import (
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
)

// Setup isolates HOME/XDG to a temp dir and forces credstore's file backend
// with a known passphrase via the §1.4 named env vars. The darwin `security`
// and Linux `secret-tool` legacy probes are neutralized so the suite is
// hermetic regardless of the destination backend (§2.3). Returns the temp
// dir so a test can plant legacy artifacts (token.json, credentials.json,
// config.json, a keychain item) under it.
func Setup(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdgconfig"))
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "file")
	t.Setenv("GOOGLE_READONLY_KEYRING_PASSPHRASE", "test-passphrase")
	t.Setenv("GRO_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN", "1")
	t.Setenv("GRO_TEST_DISABLE_LEGACY_SECRETTOOL_SCAN", "1")
	migrationsink.Reset()
	t.Cleanup(migrationsink.Reset)
	return tmp
}

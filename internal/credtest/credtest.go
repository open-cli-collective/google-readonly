// Package credtest provides a hermetic credential environment for tests
// (§1.12 test obligation). It delegates state-dir isolation to the shared
// cli-common/statedirtest helper (the full 7-var env set per §3.1 — closes
// the Windows real-dir leak the old HOME/XDG-only setup had), then layers
// the gro-specific keyring backend selection on top: credstore's
// encrypted-file backend with a known passphrase, plus the two legacy
// keychain/secret-tool scan disablers so no test ever shells out.
//
// It is deliberately a tiny leaf (only testing + statedirtest + config +
// migrationsink) so that the white-box `package keychain` tests can import it
// without the keychain<->testutil import cycle that the fixture-heavy
// internal/testutil would introduce.
package credtest

import (
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
)

// Setup isolates the full §3.1 7-var env set under t.TempDir() (via
// statedirtest.Hermetic) and forces credstore's file backend with a known
// passphrase via the §1.4 named env vars. The darwin `security` and Linux
// `secret-tool` legacy probes are neutralized so the suite is hermetic
// regardless of the destination backend (§2.3). Returns the temp root so a
// test can plant legacy artifacts (token.json, credentials.json, config.json,
// a keychain item) — but tests should resolve their paths through
// config.GetConfigDir() / ConfigDir(t) below, not by hand-building subdirs,
// because os.UserConfigDir is platform-native (macOS ~/Library/Application
// Support, Windows %APPDATA%) and not derived from any single env var.
func Setup(t *testing.T) string {
	t.Helper()
	tmp := statedirtest.Hermetic(t)
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "file")
	t.Setenv("GOOGLE_READONLY_KEYRING_PASSPHRASE", "test-passphrase")
	t.Setenv("GRO_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN", "1")
	t.Setenv("GRO_TEST_DISABLE_LEGACY_SECRETTOOL_SCAN", "1")
	migrationsink.Reset()
	t.Cleanup(migrationsink.Reset)
	return tmp
}

// ConfigDir resolves the post-statedirtest hermetic config dir and creates
// it. Tests that plant legacy artifacts in the config dir should use this
// rather than hand-building subdirs of Setup's tmp root, which only worked on
// Linux pre-MON-5371.
func ConfigDir(t *testing.T) string {
	t.Helper()
	dir, err := config.GetConfigDir()
	if err != nil {
		t.Fatalf("credtest.ConfigDir: %v", err)
	}
	return dir
}

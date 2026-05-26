package keychain

import (
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

// resetOverride keeps the package-level flag override clean across
// tests so a leaked value from one test can't tilt the next.
func resetOverride(t *testing.T) {
	t.Helper()
	SetBackendFlagOverride("", false)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })
}

// TestOpenWith_ConfigOnlyMemoryBackend proves the openWith call site
// actually consumes cfg.Keyring.Backend via BindBackendFlag. Uses the
// memory backend so the test is platform-/passphrase-free.
func TestOpenWith_ConfigOnlyMemoryBackend(t *testing.T) {
	resetOverride(t)
	// Clear the env var so flag > env > config > default precedence
	// doesn't silently route the test through the env layer.
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "")

	cfg := &config.Config{
		CredentialRef: config.DefaultCredentialRef,
		Keyring:       config.KeyringConfig{Backend: string(credstore.BackendMemory)},
	}
	st, err := openWith(cfg, false, false)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	defer func() { _ = st.Close() }()

	b, src := st.Backend()
	if b != credstore.BackendMemory {
		t.Errorf("Backend = %q, want %q", b, credstore.BackendMemory)
	}
	if src != credstore.SourceConfig {
		t.Errorf("Source = %q, want %q (config-only path should attribute to config)", src, credstore.SourceConfig)
	}
}

// TestOpenWith_FlagOverridesConfig proves --backend wins over
// keyring.backend at the openWith binding site. Sets the flag override
// to memory and the config-side to file (which would normally require
// a passphrase and fail), and asserts credstore reports the resolved
// backend as memory with SourceExplicit — proving the flag pair flowed
// through BindBackendFlag and won precedence over config.
func TestOpenWith_FlagOverridesConfig(t *testing.T) {
	resetOverride(t)
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "")
	SetBackendFlagOverride(string(credstore.BackendMemory), true)

	cfg := &config.Config{
		CredentialRef: config.DefaultCredentialRef,
		// Config-side intentionally different from the flag to prove the
		// flag wins. Memory is the only safe backend without passphrase
		// material; pick a config value that would NOT succeed in this
		// test environment (file requires a passphrase) so a regression
		// where the flag is dropped would surface as a passphrase error
		// rather than a silent success.
		Keyring: config.KeyringConfig{Backend: string(credstore.BackendFile)},
	}
	st, err := openWith(cfg, false, false)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	defer func() { _ = st.Close() }()

	b, src := st.Backend()
	if b != credstore.BackendMemory {
		t.Errorf("Backend = %q, want %q (flag should override config)", b, credstore.BackendMemory)
	}
	if src != credstore.SourceExplicit {
		t.Errorf("Source = %q, want %q", src, credstore.SourceExplicit)
	}
}

//go:build darwin && cgo

package keychain

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/byteness/keyring"
	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
)

func TestKeychainMetadataGated(t *testing.T) {
	if os.Getenv("GRO_KEYCHAIN_METADATA_TEST") != "1" {
		t.Skip("set GRO_KEYCHAIN_METADATA_TEST=1 to write to the real macOS Keychain")
	}

	home := os.Getenv("HOME")
	credtest.Setup(t)
	// credtest.Setup isolates HOME for config tests, but the real macOS
	// Keychain backend uses HOME to find the user's default login keychain.
	t.Setenv("HOME", home)
	SetBackendFlagOverride(string(credstore.BackendKeychain), true)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	profile := "metadata-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ref := "google-readonly/" + profile
	account := profile + "/" + KeyOAuthToken
	t.Logf("using synthetic Keychain ref %q account %q", ref, account)

	st, err := openWith(&config.Config{CredentialRef: ref}, false, false)
	if err != nil {
		t.Fatalf("openWith(%q): %v", ref, err)
	}
	t.Cleanup(func() { _ = st.Close() })
	t.Cleanup(func() { _ = st.DeleteToken() })

	kr, err := keyring.Open(keyring.Config{
		ServiceName:              "google-readonly",
		AllowedBackends:          []keyring.BackendType{keyring.KeychainBackend},
		KeychainTrustApplication: true,
	})
	if err != nil {
		t.Fatalf("open ByteNess Keychain backend: %v", err)
	}
	t.Cleanup(func() { _ = kr.Remove(account) })

	wantLabel := "google-readonly " + account
	wantDescription := "Credential for google-readonly " + account

	if err := st.SetToken(&oauth2.Token{AccessToken: "fresh-access", RefreshToken: "fresh-refresh"}); err != nil {
		t.Fatalf("fresh SetToken: %v", err)
	}
	assertToken(t, st, "fresh-access", "fresh-refresh")
	assertMetadata(t, kr, account, wantLabel, wantDescription)

	if err := kr.Remove(account); err != nil {
		t.Fatalf("remove fresh item before stale seed: %v", err)
	}

	const (
		staleLabel       = "stale google token"
		staleDescription = "stale metadata before repair"
	)
	if err := kr.Set(keyring.Item{
		Key:         account,
		Data:        []byte(`{"access_token":"legacy","refresh_token":"legacy-refresh"}`),
		Label:       staleLabel,
		Description: staleDescription,
	}); err != nil {
		t.Fatalf("seed stale metadata item: %v", err)
	}
	seeded, err := kr.GetMetadata(account)
	if err != nil {
		t.Fatalf("GetMetadata(%q) after seed: %v", account, err)
	}
	if seeded.Item == nil {
		t.Fatalf("GetMetadata(%q) after seed returned nil item", account)
	}
	if seeded.Label != staleLabel {
		t.Fatalf("seeded label = %q, want %q", seeded.Label, staleLabel)
	}
	if seeded.Description != staleDescription {
		t.Fatalf("seeded description = %q, want %q", seeded.Description, staleDescription)
	}

	if err := st.SetToken(&oauth2.Token{AccessToken: "repair-access", RefreshToken: "repair-refresh"}); err != nil {
		t.Fatalf("repair SetToken: %v", err)
	}
	assertToken(t, st, "repair-access", "repair-refresh")
	assertMetadata(t, kr, account, wantLabel, wantDescription)
}

func assertToken(t *testing.T, st *Store, wantAccess, wantRefresh string) {
	t.Helper()

	tok, err := st.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.AccessToken != wantAccess {
		t.Fatalf("AccessToken = %q, want %q", tok.AccessToken, wantAccess)
	}
	if tok.RefreshToken != wantRefresh {
		t.Fatalf("RefreshToken = %q, want %q", tok.RefreshToken, wantRefresh)
	}
}

func assertMetadata(t *testing.T, kr keyring.Keyring, account, wantLabel, wantDescription string) {
	t.Helper()

	md, err := kr.GetMetadata(account)
	if err != nil {
		t.Fatalf("GetMetadata(%q): %v", account, err)
	}
	if md.Item == nil {
		t.Fatalf("GetMetadata(%q) returned nil item", account)
	}
	if len(md.Data) != 0 {
		t.Fatalf("metadata Data length = %d, want 0", len(md.Data))
	}
	if md.Label != wantLabel {
		t.Fatalf("metadata label = %q, want %q", md.Label, wantLabel)
	}
	if md.Description != wantDescription {
		t.Fatalf("metadata description = %q, want %q", md.Description, wantDescription)
	}
}

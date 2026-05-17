// Package noleak holds gro's §1.12 acceptance suite: no access secret may
// appear on any output channel, and the §1.8 one-time-migration signal must
// behave exactly once (JSON splice or stderr line, never leaking the token).
package noleak

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	cfgcmd "github.com/open-cli-collective/google-readonly/internal/cmd/config"
	mecmd "github.com/open-cli-collective/google-readonly/internal/cmd/me"
	"github.com/open-cli-collective/google-readonly/internal/cmd/setcred"
	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
	"github.com/open-cli-collective/google-readonly/internal/output"
	"github.com/open-cli-collective/google-readonly/internal/people"
)

// fakePeople is a no-network mecmd.PeopleClient for the me no-leak test.
type fakePeople struct{}

func (fakePeople) GetMe(_ context.Context) (*people.Profile, error) {
	return &people.Profile{ResourceName: "people/c1", DisplayName: "Ada", PrimaryEmail: "ada@example.com"}, nil
}

const (
	access  = "xoxp-SECRET-ACCESS-TOKEN-abcdefghijklmnop"
	refresh = "1//SECRET-REFRESH-TOKEN-qrstuvwxyz0123456789"
)

// canaries are the full secrets plus prefix/suffix slices (catches a masked
// or truncated leak, not just the verbatim value).
func canaries() []string {
	return []string{
		access, access[:16], access[len(access)-8:],
		refresh, refresh[:16], refresh[len(refresh)-8:],
	}
}

const clientJSON = `{"installed":{"client_id":"123.apps.googleusercontent.com","project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","client_secret":"deployment-not-secret","redirect_uris":["http://localhost"]}}`

// seed plants a valid oauth_client.json + a keyring token whose access/refresh
// values are the canaries, and returns the config dir.
func seed(t *testing.T) string {
	t.Helper()
	tmp := credtest.Setup(t)
	dir := filepath.Join(tmp, "xdgconfig", config.DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.OAuthClientFile), []byte(clientJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetToken(&oauth2.Token{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer"}); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	return dir
}

func assertNoLeak(t *testing.T, label string, out []byte) {
	t.Helper()
	if err := credstore.NoLeakAssertion(out, canaries()...); err != nil {
		t.Fatalf("%s leaked secret material: %v\n--- output ---\n%s", label, err, out)
	}
}

func TestConfigShowNeverLeaks(t *testing.T) {
	seed(t)
	for _, args := range [][]string{
		{"show"}, {"show", "--json"}, {"show", "--verbose"}, {"show", "--json", "--verbose"},
	} {
		cmd := cfgcmd.NewCommand()
		var so, se bytes.Buffer
		cmd.SetOut(&so)
		cmd.SetErr(&se)
		cmd.SetArgs(args)
		// stdout from fmt.Printf in runShow is process stdout; capture it.
		got := captureStdout(t, func() { _ = cmd.Execute() })
		assertNoLeak(t, "config "+strings.Join(args, " ")+" [stdout]", []byte(got))
		assertNoLeak(t, "config "+strings.Join(args, " ")+" [cobra-out]", so.Bytes())
		assertNoLeak(t, "config "+strings.Join(args, " ")+" [cobra-err]", se.Bytes())
	}
}

func TestConfigClearNeverLeaks(t *testing.T) {
	for _, args := range [][]string{{"clear"}, {"clear", "--all"}, {"clear", "--dry-run"}} {
		seed(t)
		cmd := cfgcmd.NewCommand()
		cmd.SetArgs(args)
		so, se := captureBoth(t, func() { _ = cmd.Execute() })
		assertNoLeak(t, "config "+strings.Join(args, " ")+" [stdout]", []byte(so))
		assertNoLeak(t, "config "+strings.Join(args, " ")+" [stderr]", []byte(se))
	}
}

func TestSetCredentialNeverLeaks(t *testing.T) {
	credtest.Setup(t)
	blob := `{"access_token":"` + access + `","refresh_token":"` + refresh + `","token_type":"Bearer"}`
	cmd := setcred.NewCmd()
	cmd.SetArgs([]string{"--key", keychain.KeyOAuthToken, "--stdin"})
	cmd.SetIn(strings.NewReader(blob))
	so, se := captureBoth(t, func() { _ = cmd.Execute() })
	assertNoLeak(t, "set-credential --stdin [stdout]", []byte(so))
	assertNoLeak(t, "set-credential --stdin [stderr]", []byte(se))
}

// TestMeNeverLeaks covers the §445 smoke command across text + --json +
// --extended (which reads token expiry from the keyring via gatherExtras),
// on both output channels. me's People client is mocked (no network).
func TestMeNeverLeaks(t *testing.T) {
	seed(t)
	orig := mecmd.ClientFactory
	mecmd.ClientFactory = func(_ context.Context) (mecmd.PeopleClient, error) {
		return fakePeople{}, nil
	}
	t.Cleanup(func() { mecmd.ClientFactory = orig })

	for _, args := range [][]string{{}, {"--json"}, {"--extended"}, {"--extended", "--json"}} {
		cmd := mecmd.NewCommand()
		cmd.SetArgs(args)
		so, se := captureBoth(t, func() { _ = cmd.Execute() })
		label := "me " + strings.Join(args, " ")
		assertNoLeak(t, label+" [stdout]", []byte(so))
		assertNoLeak(t, label+" [stderr]", []byte(se))
	}
}

// ---- §1.8 migration signal -------------------------------------------------

func TestMigrationSignal_JSONSpliceThenConsumed(t *testing.T) {
	migrationsink.Reset()
	t.Cleanup(migrationsink.Reset)
	migrationsink.Record(credstore.NewMigrationBlock(
		credstore.MigrationJSONEntry("oauth_token", "file:/x/token.json", "keyring:google-readonly/default/oauth_token")),
		"gro: migrated oauth_token")

	var buf bytes.Buffer
	if err := output.JSON(&buf, map[string]string{"resourceName": "people/c1"}); err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("spliced output is not valid JSON: %v\n%s", err, buf.String())
	}
	if _, ok := obj["_migration"]; !ok {
		t.Fatalf("first JSON after migration must carry _migration: %s", buf.String())
	}
	if _, ok := obj["resourceName"]; !ok {
		t.Fatalf("original fields must be preserved: %s", buf.String())
	}

	// Consumed: a second JSON has no _migration and no stale state.
	var buf2 bytes.Buffer
	_ = output.JSON(&buf2, map[string]string{"resourceName": "people/c1"})
	if strings.Contains(buf2.String(), "_migration") {
		t.Fatalf("_migration must be consume-once: %s", buf2.String())
	}
	var flushed bytes.Buffer
	migrationsink.FlushMigrationNotice(&flushed)
	if flushed.Len() != 0 {
		t.Fatalf("flush after JSON-consume must be a no-op, got %q", flushed.String())
	}
}

func TestMigrationSignal_TextFlushesToStderrOnce(t *testing.T) {
	migrationsink.Reset()
	t.Cleanup(migrationsink.Reset)
	migrationsink.Record(credstore.NewMigrationBlock(
		credstore.MigrationJSONEntry("oauth_token", "file:/x/token.json", "keyring:google-readonly/default/oauth_token")),
		"gro: migrated oauth_token into keyring google-readonly/default")

	var stderr bytes.Buffer
	migrationsink.FlushMigrationNotice(&stderr)
	if !strings.Contains(stderr.String(), "migrated oauth_token") {
		t.Fatalf("text path must emit the human line: %q", stderr.String())
	}
	// Second flush is silent (consume-once).
	var again bytes.Buffer
	migrationsink.FlushMigrationNotice(&again)
	if again.Len() != 0 {
		t.Fatalf("second flush must be silent, got %q", again.String())
	}
}

// Migration succeeds, the downstream command then errors: the signal must
// still be pending so root.runRoot's deferred flush surfaces it (exercised
// via config test, which runs keychain.Open() — migrating a planted legacy
// token — then fails creating the Gmail client with no network).
func TestMigrationSignal_SurvivesDownstreamError(t *testing.T) {
	tmp := credtest.Setup(t)
	dir := filepath.Join(tmp, "xdgconfig", config.DirName)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, config.OAuthClientFile), []byte(clientJSON), 0o644)
	if err := os.WriteFile(filepath.Join(dir, "token.json"),
		[]byte(`{"access_token":"`+access+`","refresh_token":"`+refresh+`","token_type":"Bearer"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := cfgcmd.NewCommand()
	cmd.SetArgs([]string{"test"})
	out := captureStdout(t, func() { _ = cmd.Execute() }) // expected to error downstream

	// The legacy token.json was migrated and removed despite the later error.
	if _, e := os.Stat(filepath.Join(dir, "token.json")); !os.IsNotExist(e) {
		t.Fatal("legacy token.json should have been migrated/removed before the downstream failure")
	}
	// The signal is still pending → runRoot's deferred flush would surface it.
	var stderr bytes.Buffer
	migrationsink.FlushMigrationNotice(&stderr)
	if !strings.Contains(stderr.String(), "migrated oauth_token") {
		t.Fatalf("migration notice lost after downstream error: stderr=%q", stderr.String())
	}
	assertNoLeak(t, "config test [stdout]", []byte(out))
	assertNoLeak(t, "config test [migration notice]", stderr.Bytes())
}

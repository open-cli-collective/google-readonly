package keychain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
)

// validClientJSONFixture is a minimal but structurally valid Google "installed"
// (desktop-app) OAuth client JSON — enough for google.ConfigFromJSON.
const validClientJSONFixture = `{"installed":{"client_id":"123.apps.googleusercontent.com","project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","client_secret":"shhh","redirect_uris":["http://localhost"]}}`

const tokenA = `{"access_token":"AAA","refresh_token":"RRR","token_type":"Bearer"}`
const tokenB = `{"access_token":"BBB","refresh_token":"QQQ","token_type":"Bearer"}`

func testCfg() *config.Config {
	return &config.Config{CredentialRef: config.DefaultCredentialRef}
}

// ---- pure resolver (planMigration) ---------------------------------------

func TestPlanMigration(t *testing.T) {
	noTarget := func() (string, bool) { return "", false }

	t.Run("single resolvable value writes+signals+cleans", func(t *testing.T) {
		var deleted bool
		c := []candidate{{location: "file:/x/token.json", value: tokenA, deleter: func() error { deleted = true; return nil }}}
		p, err := planMigration("google-readonly", "default", "google-readonly/default", c, noTarget, false)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.write != tokenA || !p.signal || len(p.cleanups) != 1 {
			t.Fatalf("bad plan: %+v", p)
		}
		for _, d := range p.cleanups {
			_ = d()
		}
		if !deleted {
			t.Fatal("cleanup not wired")
		}
	})

	t.Run("legacy-vs-legacy disagreement always conflicts (even with overwrite)", func(t *testing.T) {
		c := []candidate{
			{location: "keychain:google-readonly/oauth_token", value: tokenA, deleter: func() error { return nil }},
			{location: "file:/x/token.json", value: tokenB, deleter: func() error { return nil }},
		}
		for _, ov := range []bool{false, true} {
			_, err := planMigration("google-readonly", "default", "google-readonly/default", c, noTarget, ov)
			if !errors.Is(err, credstore.ErrMigrationConflict) {
				t.Fatalf("overwrite=%v: want ErrMigrationConflict, got %v", ov, err)
			}
			if strings.Contains(err.Error(), "AAA") || strings.Contains(err.Error(), "BBB") {
				t.Fatalf("conflict error leaked a value: %v", err)
			}
			if !strings.Contains(err.Error(), "keychain:google-readonly/oauth_token") ||
				!strings.Contains(err.Error(), "file:/x/token.json") {
				t.Fatalf("conflict error must name every source: %v", err)
			}
		}
	})

	t.Run("legacy-vs-target disagree: conflict without overwrite, resolved with", func(t *testing.T) {
		c := []candidate{{location: "file:/x/token.json", value: tokenA, deleter: func() error { return nil }}}
		target := func() (string, bool) { return tokenB, true }

		if _, err := planMigration("s", "default", "s/default", c, target, false); !errors.Is(err, credstore.ErrMigrationConflict) {
			t.Fatalf("want conflict, got %v", err)
		}
		p, err := planMigration("s", "default", "s/default", c, target, true)
		if err != nil || p.write != tokenA || !p.signal {
			t.Fatalf("overwrite should force legacy: %+v err=%v", p, err)
		}
	})

	t.Run("zero candidates: no write, no signal, no cleanups", func(t *testing.T) {
		p, err := planMigration("s", "default", "s/default", nil, noTarget, false)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.write != "" || p.signal || len(p.cleanups) != 0 {
			t.Fatalf("empty candidates must be a pure no-op: %+v", p)
		}
	})

	t.Run("equal target is cleanup-only, no signal", func(t *testing.T) {
		c := []candidate{{location: "file:/x/token.json", value: tokenA, deleter: func() error { return nil }}}
		p, err := planMigration("s", "default", "s/default", c, func() (string, bool) { return tokenA, true }, false)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.write != "" || p.signal || len(p.cleanups) != 1 {
			t.Fatalf("equal target must be cleanup-only/no-signal: %+v", p)
		}
	})
}

// ---- Store round-trip (file backend, hermetic) ---------------------------

func TestStoreRoundTripAndDelete(t *testing.T) {
	credtest.Setup(t)
	st, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = st.Close() }()

	if h, herr := st.HasToken(); herr != nil || h {
		t.Fatalf("fresh store should have no token (has=%v err=%v)", h, herr)
	}
	if _, err := st.Token(); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("want ErrTokenNotFound, got %v", err)
	}
	want := &oauth2.Token{AccessToken: "AAA", RefreshToken: "RRR", TokenType: "Bearer"}
	if err := st.SetToken(want); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := st.Token()
	if err != nil || got.AccessToken != "AAA" || got.RefreshToken != "RRR" {
		t.Fatalf("round-trip mismatch: %+v err=%v", got, err)
	}
	if err := st.DeleteToken(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := st.DeleteToken(); err != nil { // idempotent
		t.Fatalf("second delete must be idempotent: %v", err)
	}
}

// ---- token migration matrix (file token.json) ----------------------------

func TestMigrateTokenFileAndIdempotent(t *testing.T) {
	credtest.Setup(t)
	tokenPath := filepath.Join(credtest.ConfigDir(t), "token.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenPath, []byte(tokenA), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := openWith(testCfg(), false, true) // runMigration
	if err != nil {
		t.Fatalf("migrate open: %v", err)
	}
	tok, err := st.Token()
	if err != nil || tok.AccessToken != "AAA" {
		t.Fatalf("token not migrated into keyring: %+v err=%v", tok, err)
	}
	_ = st.Close()
	if _, statErr := os.Stat(tokenPath); !os.IsNotExist(statErr) {
		t.Fatal("legacy token.json must be removed after migration")
	}

	// Second run: nothing legacy left → no-op, still readable.
	st2, err := openWith(testCfg(), false, true)
	if err != nil {
		t.Fatalf("idempotent reopen: %v", err)
	}
	defer func() { _ = st2.Close() }()
	if h, herr := st2.HasToken(); herr != nil || !h {
		t.Fatalf("token vanished on idempotent reopen (has=%v err=%v)", h, herr)
	}
}

func TestMigrateConflictFailsLoudWithoutLeaking(t *testing.T) {
	credtest.Setup(t)
	pre, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := pre.SetToken(&oauth2.Token{AccessToken: "AAA", RefreshToken: "RRR"}); err != nil {
		t.Fatal(err)
	}
	_ = pre.Close()

	tokenPath := filepath.Join(credtest.ConfigDir(t), "token.json")
	_ = os.MkdirAll(filepath.Dir(tokenPath), 0o700)
	if err := os.WriteFile(tokenPath, []byte(tokenB), 0o600); err != nil { // differs from keyring
		t.Fatal(err)
	}

	_, err = openWith(testCfg(), false, true)
	if !errors.Is(err, credstore.ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	if strings.Contains(err.Error(), "AAA") || strings.Contains(err.Error(), "BBB") ||
		strings.Contains(err.Error(), "RRR") || strings.Contains(err.Error(), "QQQ") {
		t.Fatalf("conflict error leaked token material: %v", err)
	}

	// --overwrite resolves it (forces legacy).
	st, err := openWith(testCfg(), true, true)
	if err != nil {
		t.Fatalf("overwrite migrate: %v", err)
	}
	defer func() { _ = st.Close() }()
	got, _ := st.Token()
	if got.AccessToken != "BBB" {
		t.Fatalf("overwrite did not force legacy: %+v", got)
	}
}

// TestMigrateTokenFile_OldHandRolledPath covers the MON-5371 token-source
// enumeration extension: a pre-MON-5371 macOS/Windows install can have a
// token.json at the OLD hand-rolled config dir (not the new statedir-
// resolved dir). The migrator must find it and migrate it just like a
// new-dir token.
func TestMigrateTokenFile_OldHandRolledPath(t *testing.T) {
	credtest.Setup(t)
	oldTokenPath, err := config.OldHandRolledTokenPath()
	if err != nil {
		t.Fatalf("OldHandRolledTokenPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(oldTokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldTokenPath, []byte(tokenA), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := openWith(testCfg(), false, true) // runMigration
	if err != nil {
		t.Fatalf("migrate open: %v", err)
	}
	tok, err := st.Token()
	if err != nil || tok.AccessToken != "AAA" {
		t.Fatalf("token at OLD path not migrated into keyring: %+v err=%v", tok, err)
	}
	_ = st.Close()
	if _, statErr := os.Stat(oldTokenPath); !os.IsNotExist(statErr) {
		t.Fatal("old-path token.json must be removed after migration (no stale plaintext)")
	}
}

// TestMigrateTokenFile_OldAndNewDivergent covers the §1.8 invariant: if both
// old/token.json and new/token.json exist with different bytes, the migrator
// must fail loud and mutate nothing.
func TestMigrateTokenFile_OldAndNewDivergent(t *testing.T) {
	credtest.Setup(t)
	oldTokenPath, err := config.OldHandRolledTokenPath()
	if err != nil {
		t.Fatalf("OldHandRolledTokenPath: %v", err)
	}
	newTokenPath, err := config.GetTokenPath()
	if err != nil {
		t.Fatalf("GetTokenPath: %v", err)
	}
	if oldTokenPath == newTokenPath {
		t.Skip("Linux: old and new paths identical — dedup makes divergence impossible")
	}
	if err := os.MkdirAll(filepath.Dir(oldTokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(newTokenPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldTokenPath, []byte(tokenA), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newTokenPath, []byte(tokenB), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = openWith(testCfg(), false, true)
	if err == nil {
		t.Fatal("divergent old/new token.json must fail loud, got nil")
	}
	if _, e := os.Stat(oldTokenPath); e != nil {
		t.Errorf("old token.json must remain on conflict: %v", e)
	}
	if _, e := os.Stat(newTokenPath); e != nil {
		t.Errorf("new token.json must remain on conflict: %v", e)
	}
}

// ---- deployment-material migration (credentials.json -> oauth_client.json) -

func TestMigrateOAuthClientJSON(t *testing.T) {
	t.Run("copy-only when target absent", func(t *testing.T) {
		credtest.Setup(t)
		dir := credtest.ConfigDir(t)
		_ = os.MkdirAll(dir, 0o700)
		legacy := filepath.Join(dir, "credentials.json")
		if err := os.WriteFile(legacy, []byte(validClientJSONFixture), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := testCfg()
		if err := migrateOAuthClientJSON(cfg); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if _, e := os.Stat(legacy); !os.IsNotExist(e) {
			t.Fatal("legacy credentials.json must be removed")
		}
		if b, e := os.ReadFile(cfg.OAuthClientPath); e != nil || string(b) != validClientJSONFixture {
			t.Fatalf("target not written: %v", e)
		}
		// second run is silent / no error (legacy gone)
		if err := migrateOAuthClientJSON(cfg); err != nil {
			t.Fatalf("second run must be a no-op: %v", err)
		}
	})

	t.Run("target valid present: legacy removed", func(t *testing.T) {
		credtest.Setup(t)
		dir := credtest.ConfigDir(t)
		_ = os.MkdirAll(dir, 0o700)
		legacy := filepath.Join(dir, "credentials.json")
		target := filepath.Join(dir, "oauth_client.json")
		_ = os.WriteFile(legacy, []byte(validClientJSONFixture), 0o600)
		_ = os.WriteFile(target, []byte(validClientJSONFixture), 0o644)
		cfg := &config.Config{CredentialRef: config.DefaultCredentialRef, OAuthClientPath: target}
		if err := migrateOAuthClientJSON(cfg); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if _, e := os.Stat(legacy); !os.IsNotExist(e) {
			t.Fatal("legacy must be removed when a valid target exists")
		}
	})

	t.Run("legacy valid, target invalid: target rewritten from legacy", func(t *testing.T) {
		credtest.Setup(t)
		dir := credtest.ConfigDir(t)
		_ = os.MkdirAll(dir, 0o700)
		legacy := filepath.Join(dir, "credentials.json")
		target := filepath.Join(dir, "oauth_client.json")
		_ = os.WriteFile(legacy, []byte(validClientJSONFixture), 0o600)
		_ = os.WriteFile(target, []byte("corrupt"), 0o644)
		cfg := &config.Config{CredentialRef: config.DefaultCredentialRef, OAuthClientPath: target}
		if err := migrateOAuthClientJSON(cfg); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if b, e := os.ReadFile(target); e != nil || string(b) != validClientJSONFixture {
			t.Fatalf("invalid target must be rewritten from valid legacy: %v", e)
		}
		if _, e := os.Stat(legacy); !os.IsNotExist(e) {
			t.Fatal("legacy must be removed after rewrite")
		}
	})

	t.Run("both invalid: nothing deleted, error names paths+fingerprints", func(t *testing.T) {
		credtest.Setup(t)
		dir := credtest.ConfigDir(t)
		_ = os.MkdirAll(dir, 0o700)
		legacy := filepath.Join(dir, "credentials.json")
		target := filepath.Join(dir, "oauth_client.json")
		_ = os.WriteFile(legacy, []byte("garbage"), 0o600)
		_ = os.WriteFile(target, []byte("also garbage"), 0o644)
		cfg := &config.Config{CredentialRef: config.DefaultCredentialRef, OAuthClientPath: target}
		err := migrateOAuthClientJSON(cfg)
		if err == nil {
			t.Fatal("want error when both invalid")
		}
		if _, e := os.Stat(legacy); os.IsNotExist(e) {
			t.Fatal("legacy must NOT be deleted when both invalid")
		}
		if !strings.Contains(err.Error(), "sha256:") {
			t.Fatalf("error must report fingerprints: %v", err)
		}
	})
}

package config

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/cache"
	appconfig "github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

const clientSecretSentinel = "SENTINEL-CLIENT-SECRET-7f3a"

const clientJSON = `{"installed":{"client_id":"123.apps.googleusercontent.com","project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","client_secret":"` + clientSecretSentinel + `","redirect_uris":["http://localhost"]}}`

func capture(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() { var b bytes.Buffer; _, _ = io.Copy(&b, r); done <- b.String() }()
	func() {
		// Restore even if f() panics, so a failure doesn't leave os.Stdout
		// redirected for every subsequent test in the process.
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		f()
	}()
	return <-done
}

func seedTokenAndClient(t *testing.T) string {
	t.Helper()
	tmp := credtest.Setup(t)
	dir := filepath.Join(tmp, "xdgconfig", appconfig.DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, appconfig.OAuthClientFile), []byte(clientJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetToken(&oauth2.Token{AccessToken: "A", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	return dir
}

func TestRunShowReportsState(t *testing.T) {
	seedTokenAndClient(t)

	out := capture(t, func() {
		if err := runShow(false, false); err != nil {
			t.Errorf("runShow: %v", err)
		}
	})
	for _, want := range []string{"google-readonly/default", "OAuth token:", "present", "OAuth client JSON:", "sha256:"} {
		if !strings.Contains(out, want) {
			t.Errorf("text show missing %q in:\n%s", want, out)
		}
	}

	jsonOut := capture(t, func() {
		if err := runShow(true, true); err != nil {
			t.Errorf("runShow json: %v", err)
		}
	})
	var st showStatus
	if err := json.Unmarshal([]byte(jsonOut), &st); err != nil {
		t.Fatalf("show --json not valid JSON: %v\n%s", err, jsonOut)
	}
	if !st.OAuthTokenPresent || !st.OAuthClientPresent || st.OAuthClientFingerprint == "" {
		t.Fatalf("json status wrong: %+v", st)
	}
	if !strings.HasPrefix(st.OAuthClientFingerprint, "sha256:") {
		t.Errorf("fingerprint format: %q", st.OAuthClientFingerprint)
	}
	if st.OAuthClientContents == "" || !strings.Contains(st.OAuthClientContents, "client_id") {
		t.Errorf("--verbose must inline client JSON, got %q", st.OAuthClientContents)
	}
	if strings.Contains(st.OAuthClientContents, clientSecretSentinel) {
		t.Errorf("--verbose must NOT expose client_secret; got %q", st.OAuthClientContents)
	}
	if !strings.Contains(st.OAuthClientContents, "[redacted]") {
		t.Errorf("--verbose must redact client_secret to [redacted]; got %q", st.OAuthClientContents)
	}
	if strings.Contains(jsonOut, clientSecretSentinel) {
		t.Errorf("show --json must never expose client_secret")
	}
	if strings.Contains(jsonOut, `"A"`) || strings.Contains(jsonOut, `"R"`) {
		t.Errorf("show must never include the token value")
	}
}

func TestRunClearSemantics(t *testing.T) {
	t.Run("dry-run removes nothing", func(t *testing.T) {
		seedTokenAndClient(t)
		_ = capture(t, func() { _ = runClear(false, true) })
		st, err := keychain.OpenNoMigrate()
		if err != nil {
			t.Fatalf("OpenNoMigrate: %v", err)
		}
		defer func() { _ = st.Close() }()
		if h, herr := st.HasToken(); herr != nil || !h {
			t.Fatalf("--dry-run must not remove the token (has=%v err=%v)", h, herr)
		}
	})

	t.Run("clear removes the token", func(t *testing.T) {
		seedTokenAndClient(t)
		_ = capture(t, func() { _ = runClear(false, false) })
		st, err := keychain.OpenNoMigrate()
		if err != nil {
			t.Fatalf("OpenNoMigrate: %v", err)
		}
		defer func() { _ = st.Close() }()
		if h, herr := st.HasToken(); herr != nil || h {
			t.Fatalf("clear must remove the token (has=%v err=%v)", h, herr)
		}
	})

	t.Run("--all removes config.yml", func(t *testing.T) {
		dir := seedTokenAndClient(t)
		if err := appconfig.SaveConfig(&appconfig.Config{CredentialRef: appconfig.DefaultCredentialRef}); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, appconfig.ConfigFileYAML)
		if _, err := os.Stat(cfgPath); err != nil {
			t.Fatalf("precondition: config.yml should exist: %v", err)
		}
		_ = capture(t, func() { _ = runClear(true, false) })
		if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
			t.Fatal("--all must remove config.yml")
		}
	})

	t.Run("--all removes the Drive cache (new + legacy)", func(t *testing.T) {
		seedTokenAndClient(t)
		c, err := cache.New(24)
		if err != nil {
			t.Fatalf("cache.New: %v", err)
		}
		if err := c.SetDrives([]*cache.CachedDrive{{ID: "d1", Name: "Eng"}}); err != nil {
			t.Fatalf("SetDrives: %v", err)
		}
		newDir := c.GetDir()
		legacy, err := appconfig.LegacyCacheDir()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(legacy, 0o700); err != nil {
			t.Fatal(err)
		}

		_ = capture(t, func() { _ = runClear(true, false) })

		if _, err := os.Stat(newDir); !os.IsNotExist(err) {
			t.Fatalf("--all must remove the new Drive cache dir (stat err=%v)", err)
		}
		if _, err := os.Stat(legacy); !os.IsNotExist(err) {
			t.Fatalf("--all must force-clean the legacy cache dir (stat err=%v)", err)
		}
	})

	t.Run("plain clear neither initializes nor migrates the cache", func(t *testing.T) {
		seedTokenAndClient(t)
		legacy, err := appconfig.LegacyCacheDir()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(legacy, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(legacy, cache.DrivesFile), []byte(`{"drives":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		newDir, err := appconfig.CacheDirPath()
		if err != nil {
			t.Fatal(err)
		}

		_ = capture(t, func() { _ = runClear(false, false) })

		if _, err := os.Stat(legacy); err != nil {
			t.Fatalf("plain clear must not migrate/remove the legacy cache (stat err=%v)", err)
		}
		if _, err := os.Stat(newDir); !os.IsNotExist(err) {
			t.Fatalf("plain clear must not initialize the new cache dir (stat err=%v)", err)
		}
	})

	t.Run("--dry-run --all creates and removes nothing, names the cache", func(t *testing.T) {
		seedTokenAndClient(t)
		legacy, err := appconfig.LegacyCacheDir()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(legacy, 0o700); err != nil {
			t.Fatal(err)
		}
		newDir, err := appconfig.CacheDirPath()
		if err != nil {
			t.Fatal(err)
		}

		out := capture(t, func() { _ = runClear(true, true) })

		if _, err := os.Stat(legacy); err != nil {
			t.Fatalf("--dry-run must not touch the legacy cache (stat err=%v)", err)
		}
		if _, err := os.Stat(newDir); !os.IsNotExist(err) {
			t.Fatalf("--dry-run must not create the new cache dir (stat err=%v)", err)
		}
		if !strings.Contains(out, "Drive metadata cache") {
			t.Fatalf("--dry-run --all output should name the Drive metadata cache, got:\n%s", out)
		}
	})
}

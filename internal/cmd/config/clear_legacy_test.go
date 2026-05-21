package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/credtest"
)

// withSyntheticConfigCandidates overrides configFilesForClearFn so Linux CI
// can exercise the macOS/Windows "old != new" branch with explicit paths.
// Same shape as MON-5373 nrq's credentialFileCandidates seam.
func withSyntheticConfigCandidates(t *testing.T, paths []string) {
	t.Helper()
	orig := configFilesForClearFn
	configFilesForClearFn = func() ([]string, error) { return paths, nil }
	t.Cleanup(func() { configFilesForClearFn = orig })
}

func seedFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func TestConfigClearAll_RemovesAllConfigCandidates(t *testing.T) {
	credtest.Setup(t)
	tmp := t.TempDir()
	newYML := filepath.Join(tmp, "new", "config.yml")
	newJSON := filepath.Join(tmp, "new", "config.json")
	oldYML := filepath.Join(tmp, "old", "config.yml")
	oldJSON := filepath.Join(tmp, "old", "config.json")
	for _, p := range []string{newYML, newJSON, oldYML, oldJSON} {
		seedFile(t, p, "credential_ref: foo/bar\n")
	}
	withSyntheticConfigCandidates(t, []string{newYML, newJSON, oldYML, oldJSON})

	_ = capture(t, func() {
		if err := runClear(true, false); err != nil {
			t.Fatalf("runClear: %v", err)
		}
	})

	for _, p := range []string{newYML, newJSON, oldYML, oldJSON} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("--all must remove %s (stat err=%v)", p, err)
		}
	}
}

func TestConfigClearAll_DryRunReportsAllCandidates(t *testing.T) {
	credtest.Setup(t)
	tmp := t.TempDir()
	newYML := filepath.Join(tmp, "new", "config.yml")
	oldJSON := filepath.Join(tmp, "old", "config.json")
	seedFile(t, newYML, "credential_ref: foo/bar\n")
	seedFile(t, oldJSON, `{"credential_ref":"legacy/bar"}`)
	withSyntheticConfigCandidates(t, []string{newYML, oldJSON})

	out := capture(t, func() {
		if err := runClear(true, true); err != nil {
			t.Fatalf("runClear dry-run: %v", err)
		}
	})

	for _, p := range []string{newYML, oldJSON} {
		if !strings.Contains(out, p) && !strings.Contains(out, appconfig.ShortenPath(p)) {
			t.Errorf("--dry-run --all should name %q in output, got:\n%s", p, out)
		}
		if _, err := os.Stat(p); err != nil {
			t.Errorf("--dry-run must not remove %s (stat err=%v)", p, err)
		}
	}
}

func TestConfigClearAll_LegacyAbsent_OnlyCanonical(t *testing.T) {
	credtest.Setup(t)
	tmp := t.TempDir()
	newYML := filepath.Join(tmp, "new", "config.yml")
	oldYML := filepath.Join(tmp, "old", "config.yml") // never seeded
	seedFile(t, newYML, "credential_ref: foo/bar\n")
	withSyntheticConfigCandidates(t, []string{newYML, oldYML})

	out := capture(t, func() {
		if err := runClear(true, false); err != nil {
			t.Fatalf("runClear: %v", err)
		}
	})

	if _, err := os.Stat(newYML); !os.IsNotExist(err) {
		t.Errorf("--all must remove canonical %s (stat err=%v)", newYML, err)
	}
	// Absent legacy must not produce a "Removed" line. We accept silence on
	// IsNotExist.
	if strings.Contains(out, oldYML) {
		t.Errorf("absent %s should not appear in --all output, got:\n%s", oldYML, out)
	}
}

func TestConfigFilesForClear_PathIdentityDedup(t *testing.T) {
	credtest.Setup(t)
	// Engineer oldDir == newDir by pointing XDG_CONFIG_HOME at the same root
	// the canonical resolver uses. credtest.Setup put us under a hermetic
	// statedirtest env; on Linux that already produces old==new — the test
	// is meaningful there. On macOS/Windows the resolvers diverge by design
	// and this assertion just verifies no spurious duplication when they
	// happen to coincide (the canonical paths are themselves distinct).
	paths, err := configFilesForClear()
	if err != nil {
		t.Fatalf("configFilesForClear: %v", err)
	}
	seen := map[string]int{}
	for _, p := range paths {
		seen[p]++
	}
	for p, n := range seen {
		if n != 1 {
			t.Errorf("path %s appeared %d times; want 1 (per-file dedup)", p, n)
		}
	}
	// Both forms must always be present at the canonical dir (regression pin
	// for the Codex r2 minor: canonical config.json was originally missed).
	newDir, derr := newDirFor(t)
	if derr != nil {
		t.Fatalf("newDirFor: %v", derr)
	}
	wantYML := filepath.Join(newDir, "config.yml")
	wantJSON := filepath.Join(newDir, "config.json")
	if seen[wantYML] == 0 {
		t.Errorf("candidate list missing canonical config.yml at %s; got %v", wantYML, paths)
	}
	if seen[wantJSON] == 0 {
		t.Errorf("candidate list missing canonical config.json at %s; got %v", wantJSON, paths)
	}
}

func TestConfigClearAll_BrokenKeyringOpen_StillScrubsFiles(t *testing.T) {
	credtest.Setup(t)
	tmp := t.TempDir()
	newYML := filepath.Join(tmp, "new", "config.yml")
	// Force keychain.OpenNoMigrate to fail via an unknown backend value.
	seedFile(t, newYML, "credential_ref: foo/bar\n")
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "this-backend-does-not-exist")
	withSyntheticConfigCandidates(t, []string{newYML})

	out := capture(t, func() {
		if err := runClear(true, false); err != nil {
			t.Fatalf("runClear under --all must soft-degrade on keyring open failure, got: %v", err)
		}
	})

	if _, err := os.Stat(newYML); !os.IsNotExist(err) {
		t.Errorf("--all must scrub %s even when keyring open fails (stat err=%v)", newYML, err)
	}
	if !strings.Contains(out, newYML) && !strings.Contains(out, appconfig.ShortenPath(newYML)) {
		t.Errorf("expected runClear to print Removed line for %s, got:\n%s", newYML, out)
	}
}

// TestConfigClearAll_MalformedCanonicalConfig_StillScrubsFiles pins that a
// genuinely malformed config.yml on disk does NOT block --all. The fix
// removed LoadConfig from the runClear path entirely (paths come from the
// candidate helper, not from a parsed Config), so this should be trivially
// true — the test pins it against regression.
func TestConfigClearAll_MalformedCanonicalConfig_StillScrubsFiles(t *testing.T) {
	credtest.Setup(t)
	tmp := t.TempDir()
	newYML := filepath.Join(tmp, "new", "config.yml")
	// Genuinely malformed YAML (matches the relocate_test.go fixture shape).
	seedFile(t, newYML, "[unclosed_array: yes\n")
	withSyntheticConfigCandidates(t, []string{newYML})

	_ = capture(t, func() {
		if err := runClear(true, false); err != nil {
			t.Fatalf("--all must tolerate malformed canonical YAML, got: %v", err)
		}
	})

	if _, err := os.Stat(newYML); !os.IsNotExist(err) {
		t.Errorf("--all must scrub malformed %s (stat err=%v)", newYML, err)
	}
}

// TestConfigFilesForClear_ExcludesOAuthClientJSON pins the deployment-
// material exclusion: clear --all must never list oauth_client.json
// among the files it removes.
func TestConfigFilesForClear_ExcludesOAuthClientJSON(t *testing.T) {
	credtest.Setup(t)
	paths, err := configFilesForClear()
	if err != nil {
		t.Fatalf("configFilesForClear: %v", err)
	}
	for _, p := range paths {
		if filepath.Base(p) == appconfig.OAuthClientFile {
			t.Errorf("clear --all must NOT remove the OAuth client JSON (deployment material); got %s in %v", p, paths)
		}
	}
}

func TestConfigClear_PlainStillHardFailsOnBrokenKeyring(t *testing.T) {
	credtest.Setup(t)
	t.Setenv("GOOGLE_READONLY_KEYRING_BACKEND", "this-backend-does-not-exist")

	_ = capture(t, func() {
		err := runClear(false, false)
		if err == nil {
			t.Fatal("plain `clear` (no --all) must surface a keyring open failure; got nil")
		}
	})
}

// newDirFor returns the canonical config dir as configFilesForClear would
// resolve it (via the live config.GetConfigDirNoCreate). Kept as a helper
// so tests don't have to import internal/config solely for this.
func newDirFor(t *testing.T) (string, error) {
	t.Helper()
	paths, err := configFilesForClear()
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		t.Fatal("configFilesForClear returned no paths")
	}
	return filepath.Dir(paths[0]), nil
}

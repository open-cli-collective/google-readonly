package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

// reloctest fixture: gives two distinct directories (old and new), both
// inside the hermetic temp root.
func reloctest(t *testing.T) (oldDir, newDir string) {
	t.Helper()
	root := statedirtest.Hermetic(t)
	oldDir = filepath.Join(root, "old", DirName)
	newDir = filepath.Join(root, "new", DirName)
	return oldDir, newDir
}

// detectAt exercises the pure-function core that takes an injected newDir,
// so the four cases are testable on Linux even though Linux's real
// os.UserConfigDir collapses to old==new.
func detectAt(t *testing.T, oldDir, newDir string) (SharedRelocation, error) {
	t.Helper()
	// Point XDG_CONFIG_HOME at the parent of oldDir so oldHandRolledConfigDir
	// resolves to oldDir on Linux. On macOS/Windows XDG_CONFIG_HOME is
	// ignored by the new path resolver, but the old path resolver still
	// reads it — so old/new diverge as intended.
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(oldDir))
	return detectRelocation(newDir)
}

func TestRelocate_OldOnly_CopiedAtInit(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte("credential_ref: google-readonly/old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, TokenFile), []byte(`{"access_token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, OAuthClientFile), []byte("not-secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocOldOnly || !r.CopyNeeded {
		t.Fatalf("kind=%v copyNeeded=%v, want relocOldOnly w/ copyNeeded", r.Kind, r.CopyNeeded)
	}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// config.yml and oauth_client.json copied to new.
	if _, err := os.Stat(filepath.Join(newDir, ConfigFileYAML)); err != nil {
		t.Errorf("config.yml not copied to new: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, OAuthClientFile)); err != nil {
		t.Errorf("oauth_client.json not copied to new: %v", err)
	}
	// token.json NEVER copied (handled by §1.8 migrator).
	if _, err := os.Stat(filepath.Join(newDir, TokenFile)); !os.IsNotExist(err) {
		t.Errorf("token.json must NOT be copied (it's a secret); stat err=%v", err)
	}
	// Old preserved (recovery point).
	if _, err := os.Stat(filepath.Join(oldDir, ConfigFileYAML)); err != nil {
		t.Errorf("old config.yml must remain (copy-leave-old): %v", err)
	}
	if _, err := os.Stat(filepath.Join(oldDir, TokenFile)); err != nil {
		t.Errorf("old token.json must remain (migrator handles it): %v", err)
	}
}

func TestRelocate_NewOnly_LeftUntouched(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte("credential_ref: google-readonly/new\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone || r.CopyNeeded {
		t.Fatalf("kind=%v copyNeeded=%v, want relocNone", r.Kind, r.CopyNeeded)
	}
	// Apply is a no-op; new content survives.
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(newDir, ConfigFileYAML))
	if !strings.Contains(string(data), "google-readonly/new") {
		t.Errorf("new config must be untouched, got %q", string(data))
	}
}

func TestRelocate_Equal_NoOp(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, ConfigFileYAML), []byte("credential_ref: google-readonly/same\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocBothEqual {
		t.Fatalf("kind=%v, want relocBothEqual", r.Kind)
	}
	if r.CopyNeeded {
		t.Errorf("equal case must not request a copy")
	}
}

func TestRelocate_Divergent_FailLoudNamesBothPaths_MutatesNothing(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, p := range []struct{ dir, content string }{
		{oldDir, "credential_ref: google-readonly/old\n"},
		{newDir, "credential_ref: google-readonly/new\n"},
	} {
		if err := os.MkdirAll(p.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, ConfigFileYAML), []byte(p.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Snapshot pre-detect state.
	oldBefore, _ := os.ReadFile(filepath.Join(oldDir, ConfigFileYAML))
	newBefore, _ := os.ReadFile(filepath.Join(newDir, ConfigFileYAML))

	r, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("want ErrRelocationConflict, got %v", err)
	}
	if r.Kind != relocBothDivergent {
		t.Errorf("kind=%v, want relocBothDivergent", r.Kind)
	}
	// Error names both paths.
	if !strings.Contains(err.Error(), oldDir) || !strings.Contains(err.Error(), newDir) {
		t.Errorf("error must name both paths: %v", err)
	}
	// Nothing was mutated.
	oldAfter, _ := os.ReadFile(filepath.Join(oldDir, ConfigFileYAML))
	newAfter, _ := os.ReadFile(filepath.Join(newDir, ConfigFileYAML))
	if string(oldBefore) != string(oldAfter) {
		t.Errorf("old config mutated by detect")
	}
	if string(newBefore) != string(newAfter) {
		t.Errorf("new config mutated by detect")
	}
}

func TestRelocate_MalformedOld_FailLoud(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte("not-valid-yaml: : :\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte("credential_ref: google-readonly/new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("malformed old must fail loud, got %v", err)
	}
	if !strings.Contains(err.Error(), oldDir) {
		t.Errorf("error must name the malformed file: %v", err)
	}
}

func TestRelocate_MalformedNew_FailLoud(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte("credential_ref: google-readonly/old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte("not-valid-yaml: : :\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("malformed new must fail loud, got %v", err)
	}
	if !strings.Contains(err.Error(), newDir) {
		t.Errorf("error must name the malformed file: %v", err)
	}
}

func TestRelocate_Neither_PathResolvedNotCreated(t *testing.T) {
	oldDir, newDir := reloctest(t)
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone {
		t.Errorf("kind=%v, want relocNone", r.Kind)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("detect must not create old dir; stat err=%v", err)
	}
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Errorf("detect must not create new dir; stat err=%v", err)
	}
}

func TestRelocate_OAuthClientPath_DefaultDefaultIsEqual(t *testing.T) {
	// Both sides reference their respective dir's default oauth_client.json
	// — that's a location artifact, not a user choice. Equal.
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	oldCfg := "credential_ref: google-readonly/same\noauth_client_path: " + filepath.Join(oldDir, OAuthClientFile) + "\n"
	newCfg := "credential_ref: google-readonly/same\noauth_client_path: " + filepath.Join(newDir, OAuthClientFile) + "\n"
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte(oldCfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte(newCfg), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocBothEqual {
		t.Errorf("kind=%v, want relocBothEqual (defaults differ only by dir prefix)", r.Kind)
	}
}

func TestRelocate_OAuthClientPath_ExplicitNonDefaultDivergence_FailsLoud(t *testing.T) {
	// User explicitly set oauth_client_path to a non-default location on
	// one side. That's a deliberate org override and must surface.
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	oldCfg := "credential_ref: google-readonly/same\noauth_client_path: /opt/myorg/oauth_client.json\n"
	newCfg := "credential_ref: google-readonly/same\noauth_client_path: " + filepath.Join(newDir, OAuthClientFile) + "\n"
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte(oldCfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte(newCfg), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("explicit oauth_client_path divergence must fail loud, got %v", err)
	}
}

func TestRelocate_OldOnlyConfigJSON_TriggersCopy(t *testing.T) {
	// A pre-MON-5371 install on macOS may have only legacy config.json (not
	// yet promoted to config.yml). Detection must still classify as
	// relocOldOnly so init copies the JSON into the new dir, where the
	// existing promoteLegacyConfigJSON migrator picks it up.
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFile), []byte(`{"credential_ref":"google-readonly/from-old-json"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocOldOnly || !r.CopyNeeded {
		t.Fatalf("kind=%v copy=%v, want relocOldOnly w/ copy (legacy config.json present in old)",
			r.Kind, r.CopyNeeded)
	}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, ConfigFile)); err != nil {
		t.Errorf("legacy config.json must be copied into new dir: %v", err)
	}
}

func TestRelocate_LinuxOldEqualsNew_ShortCircuits(t *testing.T) {
	// When old path resolves identical to new path (the Linux steady state),
	// detect must short-circuit to relocNone without inspecting contents.
	// Simulate by pointing both at the same dir.
	root := statedirtest.Hermetic(t)
	same := filepath.Join(root, "same")
	if err := os.MkdirAll(same, 0o700); err != nil {
		t.Fatal(err)
	}
	// Point XDG_CONFIG_HOME so oldHandRolledConfigDir resolves to <same>/<DirName>.
	t.Setenv("XDG_CONFIG_HOME", same)
	r, err := detectRelocation(filepath.Join(same, DirName))
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone {
		t.Errorf("kind=%v, want relocNone on path-identity short-circuit", r.Kind)
	}
}

func TestApplyConfigRelocation_IdempotentSkipsExistingNew(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// new already has config.yml with distinct content; copy must NOT overwrite.
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte("credential_ref: google-readonly/new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, ConfigFileYAML), []byte("credential_ref: google-readonly/old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, OAuthClientFile), []byte("not-secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := SharedRelocation{Kind: relocOldOnly, OldPath: oldDir, NewPath: newDir, CopyNeeded: true}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(newDir, ConfigFileYAML))
	if !strings.Contains(string(got), "google-readonly/new") {
		t.Errorf("existing new config.yml must not be overwritten, got %q", string(got))
	}
	// oauth_client.json (absent in new) gets copied.
	if _, err := os.Stat(filepath.Join(newDir, OAuthClientFile)); err != nil {
		t.Errorf("absent-in-new file must be copied: %v", err)
	}
}

// loadConfigForRuntimeAt is the testable seam for the runtime soft-conflict
// wrapper. It wires both detectRelocation and the YAML load through an
// injected newDir + oldDir, which is the only way to exercise the divergent
// branch on a platform where os.UserConfigDir == $XDG_CONFIG_HOME (Linux).
// Production code paths use LoadConfigForRuntime via the real resolver.
func loadConfigForRuntimeAt(t *testing.T, oldDir, newDir string) (*Config, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(oldDir))
	// Reset the warn-once before the test so a previous test's flag doesn't
	// suppress the warning we want to verify.
	reloConflictOnce = sync.Once{}
	r, err := detectRelocation(newDir)
	if err != nil && !errors.Is(err, ErrRelocationConflict) {
		return nil, err
	}
	cfg := &Config{}
	if newPath, ok := firstExistingConfig(r.NewPath); ok {
		c, lerr := loadConfigFromFile(newPath)
		if lerr != nil {
			return nil, lerr
		}
		cfg = &c
	} else if r.Kind == relocOldOnly {
		if oldPath, ok := firstExistingConfig(r.OldPath); ok {
			c, lerr := loadConfigFromFile(oldPath)
			if lerr != nil {
				return nil, lerr
			}
			cfg = &c
		}
	}
	cfg.applyDefaults()
	if errors.Is(err, ErrRelocationConflict) {
		warnReloConflictOnce(err)
		return cfg, nil
	}
	return cfg, err
}

func TestLoadConfigForRuntime_SoftConflict_ReturnsCanonical(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, p := range []struct{ dir, content string }{
		{oldDir, "credential_ref: google-readonly/old\n"},
		{newDir, "credential_ref: google-readonly/new\n"},
	} {
		if err := os.MkdirAll(p.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, ConfigFileYAML), []byte(p.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadConfigForRuntimeAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("soft-conflict must return nil error, got %v", err)
	}
	// Returns the canonical (new-dir) cfg.
	if cfg.CredentialRef != "google-readonly/new" {
		t.Errorf("soft-conflict must return new-dir cfg, got CredentialRef=%q", cfg.CredentialRef)
	}
}

func TestLoadConfigForRuntime_NoConflict_PassesThrough(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, ConfigFileYAML), []byte("credential_ref: google-readonly/only-new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigForRuntimeAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if cfg.CredentialRef != "google-readonly/only-new" {
		t.Errorf("got %q, want google-readonly/only-new", cfg.CredentialRef)
	}
}

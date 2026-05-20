package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// ErrRelocationConflict is returned by LoadConfig (and surfaced through
// LoadConfigForRuntime) when both the old hand-rolled config dir and the new
// statedir-resolved config dir contain a config.yml with materially different
// user settings. Mutation-free: nothing is copied, nothing is overwritten.
// The user reconciles by running `gro init` (which fails the same way at its
// pre-write gate) or by manually deleting one side.
var ErrRelocationConflict = errors.New("config: shared old/new config diverge")

// relocKind is the four-way classification used by the relocation detector.
// Linux always collapses to relocNone because old and new paths are identical
// (os.UserConfigDir on Linux ≡ $XDG_CONFIG_HOME else $HOME/.config).
type relocKind int

const (
	relocNone          relocKind = iota // old path absent OR old==new (Linux short-circuit)
	relocOldOnly                        // only the old hand-rolled config.yml exists
	relocBothEqual                      // both exist with materially-equal Configs
	relocBothDivergent                  // both exist with materially-different Configs
)

// SharedRelocation is the result of DetectConfigRelocation. Paths are filled
// even on relocNone so callers can log/diagnose; CopyNeeded is true iff a
// gated ApplyConfigRelocation would actually do work.
type SharedRelocation struct {
	Kind       relocKind
	OldPath    string // old hand-rolled config dir; "" on Linux short-circuit
	NewPath    string // statedir-resolved config dir (always set)
	CopyNeeded bool   // relocOldOnly only
}

// oldHandRolledConfigDir reproduces the prior pre-MON-5371 resolver:
// $XDG_CONFIG_HOME if set, else $HOME/.config; then "/google-readonly". Same
// shape on Linux/macOS/Windows (the deliberate "no %APPDATA% branch"). A
// missing HOME is an error (matches the original behavior).
func oldHandRolledConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, DirName), nil
}

// OldHandRolledTokenPath is the pre-MON-5371 legacy token.json location.
// Exported so keychain.migrate's token-source enumeration can probe it with
// full §1.8 conflict semantics (per the MON-5371 plan: token.json is excluded
// from ApplyConfigRelocation and handled exclusively through the existing
// migrator).
func OldHandRolledTokenPath() (string, error) {
	dir, err := oldHandRolledConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, TokenFile), nil
}

// OldHandRolledLegacyCacheDir is the pre-B2b cache subdir under the pre-
// MON-5371 hand-rolled config dir (a real artifact on macOS/Windows installs
// that pre-date both B2b and MON-5371). cli-common does NOT resolve here,
// so internal/cache's migrator probes both this path and the in-resolver
// LegacyCacheDir, byte-carrying whichever is present. On Linux this resolves
// identical to LegacyCacheDir and dedupes inside the migrator.
func OldHandRolledLegacyCacheDir() (string, error) {
	dir, err := oldHandRolledConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, legacyCacheSubdir), nil
}

// DetectConfigRelocation classifies the old/new pair without touching disk
// beyond stats and reads. Never copies, never writes. On Linux (old==new) it
// short-circuits to relocNone. On macOS/Windows it returns one of the four
// kinds and the named paths.
func DetectConfigRelocation() (SharedRelocation, error) {
	newDir, err := configDirPath()
	if err != nil {
		return SharedRelocation{}, err
	}
	return detectRelocation(newDir)
}

// detectRelocation is the testable core: the new-dir path is injected so
// macOS/Windows divergence can be exercised on Linux CI (the same pattern
// MON-5370 used).
func detectRelocation(newDir string) (SharedRelocation, error) {
	oldDir, err := oldHandRolledConfigDir()
	if err != nil {
		// Old path unresolvable (no HOME): treat as relocNone with new-only.
		// LoadConfig still works against new; the gate is harmless.
		return SharedRelocation{Kind: relocNone, NewPath: newDir}, nil
	}
	if oldDir == newDir {
		// Linux: identical paths — nothing to relocate.
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	}

	// Both .yml and the legacy .json count as a present config. A
	// pre-MON-5371 macOS/Windows user could be on either form; the wholesale
	// copy at init carries either over, and the per-file migrator
	// (promoteLegacyConfigJSON) takes it from there.
	oldPath, oldPresent := firstExistingConfig(oldDir)
	newPath, newPresent := firstExistingConfig(newDir)
	switch {
	case !oldPresent && !newPresent:
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	case oldPresent && !newPresent:
		// Validate old-only parses cleanly BEFORE signaling CopyNeeded —
		// otherwise init's ApplyConfigRelocation would propagate a malformed
		// legacy file into the new dir, then LoadConfig would die parsing it
		// post-relocation. §3.2 malformed-old: fail loud, mutate nothing.
		if _, oerr := loadConfigFromFile(oldPath); oerr != nil {
			return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
				fmt.Errorf("%w: old %s is malformed: %w", ErrRelocationConflict, oldPath, oerr)
		}
		return SharedRelocation{Kind: relocOldOnly, OldPath: oldDir, NewPath: newDir, CopyNeeded: true}, nil
	case !oldPresent && newPresent:
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	}

	// Both present — load and compare comparable subset.
	oldCfg, oerr := loadConfigFromFile(oldPath)
	newCfg, nerr := loadConfigFromFile(newPath)
	if oerr != nil {
		return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
			fmt.Errorf("%w: old %s unreadable: %w", ErrRelocationConflict, oldPath, oerr)
	}
	if nerr != nil {
		return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
			fmt.Errorf("%w: new %s unreadable: %w", ErrRelocationConflict, newPath, nerr)
	}
	if configsMaterialEqual(oldCfg, newCfg, oldDir, newDir) {
		return SharedRelocation{Kind: relocBothEqual, OldPath: oldDir, NewPath: newDir}, nil
	}
	return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
		fmt.Errorf("%w: old %s and new %s have different settings; reconcile (or delete one) before running gro init",
			ErrRelocationConflict, oldPath, newPath)
}

// loadConfigFromFile parses a single config file (yaml or, by extension, the
// legacy json file). Defaults are NOT applied — we want the user-authored
// content for equality.
func loadConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path composed from validated config dir
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	switch filepath.Ext(path) {
	case ".json":
		if uerr := json.Unmarshal(data, &cfg); uerr != nil {
			return Config{}, uerr
		}
	default:
		if uerr := yaml.Unmarshal(data, &cfg); uerr != nil {
			return Config{}, uerr
		}
	}
	return cfg, nil
}

// configsMaterialEqual compares the user-meaningful subset of two Configs.
// CredentialRef / GrantedScopes / Keyring.Backend are real user settings and
// any difference is a divergence. OAuthClientPath is canonicalized through
// oauthClientPathEquiv so that two configs whose paths happen to be each
// dir's default (`<configdir>/oauth_client.json`) are treated as equal —
// that's a location artifact, not a user choice — while an explicit
// non-default path that differs between the two sides is a real divergence
// and must fail loud (an org override the user set deliberately).
func configsMaterialEqual(a, b Config, oldDir, newDir string) bool {
	if a.CredentialRef != b.CredentialRef {
		return false
	}
	// reflect.DeepEqual on the whole Keyring sub-struct: a new field added
	// to KeyringConfig (e.g. a passphrase env-var name) must not silently
	// classify as "same" just because the old code path only knew about
	// Backend.
	if !reflect.DeepEqual(a.Keyring, b.Keyring) {
		return false
	}
	if !slicesEqualSorted(a.GrantedScopes, b.GrantedScopes) {
		return false
	}
	if !oauthClientPathEquiv(a.OAuthClientPath, b.OAuthClientPath, oldDir, newDir) {
		return false
	}
	return true
}

// oauthClientPathEquiv treats "both empty", "both equal to their own dir's
// default oauth_client.json", and "both literally equal" as equivalent. Any
// other combination — including one side at its default and the other at a
// non-default explicit path — is a divergence the user set deliberately and
// must surface.
func oauthClientPathEquiv(aPath, bPath, aDir, bDir string) bool {
	// Normalize tilde and relative components on both sides before comparing
	// (loadConfigFromFile deliberately skips applyDefaults / ExpandPath so
	// that we compare user-authored bytes — but a `~/oauth_client.json` and
	// the same path fully expanded should not look divergent).
	aPath = ExpandPath(aPath)
	bPath = ExpandPath(bPath)
	if aPath == bPath {
		return true
	}
	aIsDefault := aPath == "" || aPath == filepath.Join(aDir, OAuthClientFile)
	bIsDefault := bPath == "" || bPath == filepath.Join(bDir, OAuthClientFile)
	return aIsDefault && bIsDefault
}

func slicesEqualSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	return reflect.DeepEqual(aa, bb)
}

// ApplyConfigRelocation copies every file in the old config dir to the new
// dir EXCEPT token.json (the only access secret; handled by keychain.migrate
// via §1.8 conflict semantics — see MON-5371 plan). Idempotent: if the new
// dir already has a same-named file, that one is left untouched. Files are
// written via temp+rename at 0600 under a 0700 dir. The old dir is not
// modified — leave-old gives the user a recovery point and matches the
// MON-5370 family pattern.
func ApplyConfigRelocation(r SharedRelocation) error {
	if !r.CopyNeeded {
		return nil
	}
	if r.OldPath == "" || r.NewPath == "" {
		return fmt.Errorf("config: ApplyConfigRelocation called with empty path")
	}
	if err := os.MkdirAll(r.NewPath, DirPerm); err != nil {
		return fmt.Errorf("creating new config dir: %w", err)
	}
	entries, err := os.ReadDir(r.OldPath)
	if err != nil {
		return fmt.Errorf("reading old config dir %s: %w", r.OldPath, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue // pre-B2b cache subdir etc.; not part of config relocation
		}
		if e.Name() == TokenFile {
			continue // secret — handled by keychain.migrate
		}
		dst := filepath.Join(r.NewPath, e.Name())
		if fileExists(dst) {
			continue // idempotent on re-run
		}
		src := filepath.Join(r.OldPath, e.Name())
		if err := copyFileAtomic(src, dst); err != nil {
			return fmt.Errorf("copying %s → %s: %w", src, dst, err)
		}
	}
	return nil
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // path from old config dir
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, TokenPerm); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// firstExistingConfig returns (<dir>/config.yml, true) if the YAML is present,
// else (<dir>/config.json, true) if only the legacy JSON is present, else
// ("", false). Mirrors LoadConfig's read-priority (YAML wins).
func firstExistingConfig(dir string) (string, bool) {
	if p := filepath.Join(dir, ConfigFileYAML); fileExists(p) {
		return p, true
	}
	if p := filepath.Join(dir, ConfigFile); fileExists(p) {
		return p, true
	}
	return "", false
}

// LoadConfigForRuntime is the soft-conflict variant of LoadConfig for non-init
// callers. On ErrRelocationConflict it prints a one-shot stderr warning, then
// returns the canonical (new-dir) config so the command can keep working —
// BUT only when a canonical config was actually read. If LoadConfig couldn't
// populate cfg (e.g. malformed YAML/JSON on the canonical side), the runtime
// must hard-fail instead of warning-and-defaulting, otherwise it would
// silently swap CredentialRef etc. back to defaults and mask the corrupt
// file. Init uses strict LoadConfig (fail-loud) at its relocation gate.
func LoadConfigForRuntime() (*Config, error) {
	cfg, err := LoadConfig()
	if err != nil && errors.Is(err, ErrRelocationConflict) && cfg != nil {
		warnReloConflictOnce(err)
		return cfg, nil
	}
	return cfg, err
}

var reloConflictOnce sync.Once

func warnReloConflictOnce(err error) {
	reloConflictOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "warning: %v; using the new config. Run `gro init` to reconcile.\n", err)
	})
}

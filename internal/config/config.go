// Package config provides centralized configuration management for the application.
// It has no external dependency on other internal packages to avoid import cycles.
//
// Per the Open CLI Collective Secret-Handling Standard §1.2 / §2.3, no access
// secret is ever written here. The per-user OAuth token lives only in the OS
// keyring (via cli-common's credstore). The OAuth client JSON is deployment
// material (§1.2) and lives in a plain file referenced by oauth_client_path.
// This file owns config.yml: the authoritative credential_ref (§1.3), the
// OAuth client JSON path, the optional §1.4 file-backend opt-in, and the
// pre-existing non-secret tuning (cache TTL, granted scopes).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-cli-collective/cli-common/statedir"
	"gopkg.in/yaml.v3"
)

const (
	// DirName is the name of the configuration directory.
	DirName = "google-readonly"
	// CredentialsFile is the legacy OAuth client JSON filename (deployment
	// material). Superseded by OAuthClientFile; retained so the one-time
	// migration can find and relocate it.
	CredentialsFile = "credentials.json"
	// OAuthClientFile is the post-migration OAuth client JSON filename
	// (deployment material per §1.2 — not a secret, not in the keyring).
	OAuthClientFile = "oauth_client.json"
	// TokenFile is the legacy OAuth token fallback filename. Superseded by
	// the keyring; retained so the one-time migration can find it.
	TokenFile = "token.json"
	// ConfigFile is the legacy JSON config filename, read once for
	// transparent upgrade to ConfigFileYAML.
	ConfigFile = "config.json"
	// ConfigFileYAML is the authoritative config filename.
	ConfigFileYAML = "config.yml"

	// DefaultCredentialRef applies when config.yml is absent or omits
	// credential_ref. Callers still resolve it via credstore.ParseRef — the
	// service/profile are never assumed structurally (§1.3).
	DefaultCredentialRef = "google-readonly/default"
)

// File and directory permission constants for consistent security settings.
const (
	// DirPerm is the permission for config directories (owner rwx only).
	DirPerm = 0700
	// TokenPerm is the permission for the config file (owner rw only). The
	// config holds no secret, but there is no reason for it to be world
	// readable.
	TokenPerm = 0600
	// OutputDirPerm is the permission for output directories.
	OutputDirPerm = 0755
	// OutputFilePerm is the permission for output files, and for the OAuth
	// client JSON (deployment material — non-secret, org-internal).
	OutputFilePerm = 0644
)

// Config is google-readonly's config.yml. Everything here is safe for an org
// to ship via MDM (§1.2); none of it is an access secret. JSON tags are
// retained so a legacy config.json is read transparently for one upgrade.
//
// The pre-MON-5371 `cache_ttl_hours` field is gone — cache TTL is now
// hard-coded per resource per cli-common/docs/working-with-state.md §4.4. An
// older config.yml that still contains `cache_ttl_hours: N` continues to
// load cleanly (yaml.v3 silently ignores unknown fields); the value is just
// inert post-port.
type Config struct {
	// CredentialRef is the authoritative <service>/<profile> keyring ref
	// (§1.3). Resolved via credstore.ParseRef; never hard-coded.
	CredentialRef string `yaml:"credential_ref" json:"credential_ref,omitempty"`
	// OAuthClientPath is the absolute path to the OAuth client JSON
	// (deployment material). Stored expanded + absolute; `~` is display-only
	// via ShortenPath. An org may override the default location here.
	OAuthClientPath string `yaml:"oauth_client_path" json:"oauth_client_path,omitempty"`
	// GrantedScopes is preserved: detects when a token's scopes drift from
	// what init granted. Not a secret.
	GrantedScopes []string `yaml:"granted_scopes,omitempty" json:"granted_scopes,omitempty"`
	// Keyring carries the optional §1.4 explicit file-backend opt-in.
	Keyring KeyringConfig `yaml:"keyring,omitempty" json:"-"`
}

// KeyringConfig is the §1.4 backend selector. Backend == "file" forces the
// encrypted-file backend; empty means OS default selection (fail-closed on
// Linux when no Secret Service is available).
type KeyringConfig struct {
	Backend string `yaml:"backend,omitempty" json:"-"`
}

// legacyCacheSubdir is the pre-B2b cache location: a "cache" subdir inside the
// config dir. Retained only so the one-time relocation can find and remove it.
// A local literal (not cache.CacheDir) — internal/config must not import
// internal/cache.
const legacyCacheSubdir = "cache"

// configScope is the cli-common state-scope for gro's config dir. The scope
// name is gro's `DirName`; the resolver returns the native per-OS user config
// dir (Linux $XDG_CONFIG_HOME or ~/.config; macOS ~/Library/Application
// Support; Windows %APPDATA%) plus that scope. A relative $XDG_CONFIG_HOME on
// Linux now yields an error (intentional tightening per
// cli-common/docs/working-with-state.md §1.1).
var configScope = statedir.Scope{Name: DirName}

// configDirPath resolves the configuration directory WITHOUT creating it.
// Delegated to cli-common/statedir so the per-OS dir is native everywhere.
func configDirPath() (string, error) {
	return configScope.ConfigDir()
}

// GetConfigDir returns the configuration directory, creating it if needed.
func GetConfigDir() (string, error) {
	return configScope.ConfigDirEnsured()
}

// CacheDirPath resolves the OS-designated cache directory WITHOUT creating it
// (used by `config clear --all --dry-run` and tests). os.UserCacheDir gives
// the canonical per-OS root: Linux $XDG_CACHE_HOME or ~/.cache, macOS
// ~/Library/Caches, Windows %LocalAppData%. We append only DirName — no
// platform-specific suffix — to keep all three consistent.
func CacheDirPath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, DirName), nil
}

// GetCacheDir returns the cache directory, creating it if needed.
func GetCacheDir() (string, error) {
	cacheDir, err := CacheDirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, DirPerm); err != nil { //nolint:gosec // not user-controlled
		return "", err
	}
	return cacheDir, nil
}

// LegacyCacheDir resolves the pre-B2b cache directory (a "cache" subdir of the
// config dir) WITHOUT creating anything — for the one-time relocation only.
func LegacyCacheDir() (string, error) {
	configDir, err := configDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, legacyCacheSubdir), nil
}

// GetCredentialsPath returns the path to the LEGACY credentials.json
// (deployment material). Used by the one-time migration to relocate it to
// OAuthClientPath; not the runtime client-JSON source post-migration.
func GetCredentialsPath() (string, error) { return inDir(CredentialsFile) }

// GetTokenPath returns the path to the LEGACY token.json fallback. Used only
// by the one-time migration into the keyring.
func GetTokenPath() (string, error) { return inDir(TokenFile) }

// GetConfigPath returns the authoritative config file path (config.yml).
func GetConfigPath() (string, error) { return inDir(ConfigFileYAML) }

// GetConfigPathNoCreate is GetConfigPath WITHOUT creating the config dir —
// for side-effect-free paths such as `config clear --dry-run`.
func GetConfigPathNoCreate() (string, error) {
	dir, err := configDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileYAML), nil
}

// LegacyConfigJSONPath returns the pre-migration config.json path.
func LegacyConfigJSONPath() (string, error) { return inDir(ConfigFile) }

// DefaultOAuthClientPath is the expanded absolute default for
// OAuthClientPath: <configdir>/oauth_client.json.
func DefaultOAuthClientPath() (string, error) { return inDir(OAuthClientFile) }

func inDir(name string) (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// ExpandPath resolves a leading ~ and makes the path absolute. Stored config
// values are always expanded; ~ is for display only (ShortenPath).
func ExpandPath(p string) string {
	if p == "" {
		return p
	}
	if p == "~" || (len(p) >= 2 && p[:2] == "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// ShortenPath replaces the home directory prefix with ~ for display, so
// errors and `config show` do not expose usernames.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) >= len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}

// LoadConfig loads config.yml. The strict variant — used by `gro init`'s
// relocation gate and by tests. Returns ErrRelocationConflict (with a wrapped
// detail message) when both the old hand-rolled and new statedir-resolved
// dirs contain materially-different config.yml files; on conflict, the
// canonical new-dir config is still returned alongside the error so callers
// can choose to soft-degrade. Runtime call sites should use
// LoadConfigForRuntime instead — see relocate.go.
//
// If new/config.yml is absent but old-only is present (the MON-5371
// macOS/Windows pre-init steady state), the old file is transparently read.
// If neither YAML is present, a legacy config.json is read once at the new
// dir (post-init); if that is also absent, defaults are returned.
//
// Defaults are always applied to the returned *Config.
func LoadConfig() (*Config, error) {
	relErr := error(nil)
	reloc, derr := DetectConfigRelocation()
	if derr != nil && errors.Is(derr, ErrRelocationConflict) {
		// Both-present-divergent — still return the canonical new-dir cfg so
		// callers can soft-degrade via LoadConfigForRuntime.
		relErr = derr
	} else if derr != nil && !errors.Is(derr, ErrRelocationConflict) {
		return nil, derr
	}

	// Priority: new/config.yml → old/config.yml (only if new is absent) →
	// legacy config.json at the new dir.
	cfg := &Config{}
	read := false
	if reloc.NewPath != "" {
		newYML := filepath.Join(reloc.NewPath, ConfigFileYAML)
		if data, err := os.ReadFile(newYML); err == nil { //nolint:gosec // path from validated dir
			if uerr := yaml.Unmarshal(data, cfg); uerr != nil {
				return nil, fmt.Errorf("parse config %s: %w", newYML, uerr)
			}
			read = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config %s: %w", newYML, err)
		}
	}
	if !read && reloc.Kind == relocOldOnly && reloc.OldPath != "" {
		oldYML := filepath.Join(reloc.OldPath, ConfigFileYAML)
		if data, err := os.ReadFile(oldYML); err == nil { //nolint:gosec // path from hand-rolled legacy dir
			if uerr := yaml.Unmarshal(data, cfg); uerr != nil {
				return nil, fmt.Errorf("parse config %s: %w", oldYML, uerr)
			}
			read = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config %s: %w", oldYML, err)
		}
	}
	if !read {
		if jerr := loadLegacyJSON(cfg); jerr != nil {
			return nil, jerr
		}
	}

	cfg.applyDefaults()
	return cfg, relErr
}

// loadLegacyJSON reads a pre-migration config.json if present (absent is not
// an error — that is the fresh-install steady state).
func loadLegacyJSON(cfg *Config) error {
	jsonPath, err := LegacyConfigJSONPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(jsonPath) //nolint:gosec // path from config dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read legacy config %s: %w", jsonPath, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse legacy config %s: %w", jsonPath, err)
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.CredentialRef == "" {
		c.CredentialRef = DefaultCredentialRef
	}
	if c.OAuthClientPath == "" {
		if p, err := DefaultOAuthClientPath(); err == nil {
			c.OAuthClientPath = p
		}
	} else {
		c.OAuthClientPath = ExpandPath(c.OAuthClientPath)
	}
}

// SaveConfig writes config.yml at 0600 under a 0700 directory using an atomic
// temp-file-in-same-dir → chmod 0600 → rename (§3 standard). The unique temp
// name from os.CreateTemp means a same-process concurrent save and a
// crash-leftover from a prior run can never collide; a hard-crash orphan tmp
// is harmless (never read as config). OAuthClientPath is persisted expanded +
// absolute so os.ReadFile never sees a literal ~.
func SaveConfig(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	// Serialize a copy: persisting an expanded path must not mutate the
	// caller's *Config (a caller inspecting OAuthClientPath after SaveConfig
	// would otherwise observe an unexpectedly rewritten value).
	out := *cfg
	if out.OAuthClientPath != "" {
		out.OAuthClientPath = ExpandPath(out.OAuthClientPath)
	}
	data, err := yaml.Marshal(&out)
	if err != nil {
		return err
	}

	final := filepath.Join(dir, ConfigFileYAML)
	tmp, err := os.CreateTemp(dir, "config-*.yml.tmp")
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp config file: %w", err)
	}
	if err := os.Chmod(tmpPath, TokenPerm); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("setting config file mode: %w", err)
	}
	if err := os.Rename(tmpPath, final); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("finalizing config file: %w", err)
	}
	return nil
}

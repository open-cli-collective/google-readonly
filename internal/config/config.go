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
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	// DefaultCacheTTLHours is the default cache TTL in hours.
	DefaultCacheTTLHours = 24
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
type Config struct {
	// CredentialRef is the authoritative <service>/<profile> keyring ref
	// (§1.3). Resolved via credstore.ParseRef; never hard-coded.
	CredentialRef string `yaml:"credential_ref" json:"credential_ref,omitempty"`
	// OAuthClientPath is the absolute path to the OAuth client JSON
	// (deployment material). Stored expanded + absolute; `~` is display-only
	// via ShortenPath. An org may override the default location here.
	OAuthClientPath string `yaml:"oauth_client_path" json:"oauth_client_path,omitempty"`
	// CacheTTLHours is preserved pre-existing tuning (not a secret).
	CacheTTLHours int `yaml:"cache_ttl_hours" json:"cache_ttl_hours"`
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

// GetConfigDir returns the configuration directory, creating it if needed.
// Uses $XDG_CONFIG_HOME if set, else ~/.config/google-readonly. Identical on
// Linux, macOS, and Windows — matches the released layout (no %APPDATA%
// branch), so config.yml sits beside the legacy files it supersedes.
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, DirName)
	if err := os.MkdirAll(configDir, DirPerm); err != nil { //nolint:gosec // not user-controlled
		return "", err
	}
	return configDir, nil
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

// LoadConfig loads config.yml. If config.yml is absent but a legacy
// config.json exists, it is read transparently (the user keeps working); the
// one-time migration rewrites it as config.yml and removes the JSON. An
// absent config entirely yields defaults. Defaults are always applied.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	ymlPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(ymlPath) //nolint:gosec // path from config dir
	switch {
	case err == nil:
		if uerr := yaml.Unmarshal(data, cfg); uerr != nil {
			return nil, fmt.Errorf("parse config %s: %w", ymlPath, uerr)
		}
	case os.IsNotExist(err):
		if jerr := loadLegacyJSON(cfg); jerr != nil {
			return nil, jerr
		}
	default:
		return nil, fmt.Errorf("read config %s: %w", ymlPath, err)
	}

	cfg.applyDefaults()
	return cfg, nil
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
	if c.CacheTTLHours <= 0 {
		c.CacheTTLHours = DefaultCacheTTLHours
	}
}

// SaveConfig writes config.yml at 0600 under a 0700 directory. OAuthClientPath
// is persisted expanded + absolute so os.ReadFile never sees a literal ~.
func SaveConfig(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if cfg.OAuthClientPath != "" {
		cfg.OAuthClientPath = ExpandPath(cfg.OAuthClientPath)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ConfigFileYAML), data, TokenPerm)
}

// GetCacheTTL returns the configured cache TTL duration.
func GetCacheTTL() time.Duration {
	cfg, err := LoadConfig()
	if err != nil {
		return time.Duration(DefaultCacheTTLHours) * time.Hour
	}
	return time.Duration(cfg.CacheTTLHours) * time.Hour
}

// GetCacheTTLHours returns the configured cache TTL in hours.
func GetCacheTTLHours() int {
	cfg, err := LoadConfig()
	if err != nil {
		return DefaultCacheTTLHours
	}
	return cfg.CacheTTLHours
}

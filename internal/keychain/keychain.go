// Package keychain is gro's credential adapter. Despite the historical
// package name, it no longer shells out to macOS `security` or Linux
// `secret-tool`, and no longer writes a plaintext token.json: it is a thin
// wrapper over cli-common's credstore, which owns OS-keyring storage,
// §1.4 backend selection (incl. Linux fail-closed and the encrypted-file
// fallback), Windows Credential Manager, and the §1.5.2 allowed-key
// allowlist. The name is retained to avoid churning every importer during
// the Phase B migration (Open CLI Collective Secret-Handling Standard §2.3).
//
// The access secret is the per-user OAuth token: the whole oauth2.Token
// (AccessToken AND RefreshToken are secret) serialized as one credstore
// string value under the single bundle key "oauth_token". The OAuth client
// JSON is deployment material (§1.2) and is NOT stored here.
package keychain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

// KeyOAuthToken is gro's single bundle key (§1.3). The migration renames the
// historical keychain account "oauth_token" into this same key under the
// resolved credential_ref.
const KeyOAuthToken = "oauth_token" //nolint:gosec // G101: a bundle key name, not a credential

// allowedKeys is gro's §1.5.2 allowlist: exactly the one bundle key.
var allowedKeys = []string{KeyOAuthToken}

// ErrTokenNotFound indicates no token exists in the keyring (errors.Is-able
// wrapper of credstore.ErrNotFound). Name retained for existing callers.
var ErrTokenNotFound = errors.New("no token found in secure storage")

// Store is an open handle to gro's credential bundle. Construct with one of
// the Open* functions, always Close. It carries the resolved ref so callers
// can report it in `config show` / errors without re-deriving it (the ref is
// not secret — §1.12).
type Store struct {
	cs      *credstore.Store
	service string
	profile string
	ref     string
}

// Open resolves the authoritative credential_ref from config.yml (§1.3 — the
// service/profile are parsed, never assumed), opens the backing credstore,
// and runs the one-time legacy migration (§1.8) before returning. Used by all
// real API commands AND `config test` (the smoke check must surface
// migration/conflicts exactly as a real command would). A legacy-vs-keyring
// conflict surfaces here as a §1.8 error.
func Open() (*Store, error) { return open(false, true) }

// OpenForMigrationOverwrite is Open with the §1.8 `--overwrite` resolution: a
// legacy value is forced over an existing keyring entry. It still cannot
// resolve a legacy-vs-legacy disagreement (the user must pick).
func OpenForMigrationOverwrite() (*Store, error) { return open(true, true) }

// OpenNoMigrate opens WITHOUT the one-time migration. Reserved for the
// diagnostic/remediation paths (`config show`, `config clear`): if migration
// ran first it would return a §1.8 conflict before the user could inspect or
// clear the conflicting entry, leaving no way out.
func OpenNoMigrate() (*Store, error) { return open(false, false) }

func open(overwrite, runMigration bool) (*Store, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	return openWith(cfg, overwrite, runMigration)
}

// OpenRef opens a store against an explicit ref instead of config.yml's
// credential_ref — used by `gro set-credential --ref` and the refresh
// persister. An empty ref falls back to the configured/default ref.
// Migration does NOT run here: the one-time §1.8 migration only ever targets
// the canonical configured ref (running it against an arbitrary --ref would
// discover the default ref's legacy data and could write it under the wrong
// service/profile).
func OpenRef(ref string) (*Store, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	if ref != "" {
		cfg.CredentialRef = ref
	}
	return openWith(cfg, false, false)
}

// openWith is the seam unit tests drive with an injected config (file-backend
// opt-in via Keyring.Backend) so they never touch a real keyring (§1.12 test
// obligation, and hermeticity).
func openWith(cfg *config.Config, overwrite, runMigration bool) (*Store, error) {
	service, profile, err := credstore.ParseRef(cfg.CredentialRef)
	if err != nil {
		return nil, fmt.Errorf("invalid credential_ref %q: %w", cfg.CredentialRef, err)
	}

	opts := &credstore.Options{AllowedKeys: allowedKeys}
	switch b := strings.TrimSpace(cfg.Keyring.Backend); b {
	case "":
		// Auto-select per §1.4 (credstore decides; fail-closed on Linux).
	case "file":
		opts.ConfigBackend = credstore.BackendFile
	default:
		// Fail closed: an unrecognized backend must not silently degrade to
		// auto-selection and store credentials somewhere unintended.
		return nil, fmt.Errorf("invalid keyring.backend %q in config (only \"file\" is supported)", b)
	}
	opts.FilePassphrase = passphraseFunc(service)

	cs, err := credstore.Open(service, opts)
	if err != nil {
		return nil, err
	}

	s := &Store{cs: cs, service: service, profile: profile, ref: cfg.CredentialRef}

	if runMigration {
		if err := migrateLegacyOverwrite(s, cfg, overwrite); err != nil {
			_ = cs.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close releases the backing store. Safe on a nil receiver.
func (s *Store) Close() error {
	if s == nil || s.cs == nil {
		return nil
	}
	return s.cs.Close()
}

// Ref returns the resolved credential ref (non-secret; safe to display).
func (s *Store) Ref() string { return s.ref }

// Service returns the resolved service segment (non-secret; used for the
// §1.4 passphrase-source label).
func (s *Store) Service() string { return s.service }

// Backend reports the credstore backend and how it was selected, for
// `config show` (§1.6). Neither value is secret.
func (s *Store) Backend() (credstore.Backend, credstore.Source) { return s.cs.Backend() }

// Token returns the OAuth token from the keyring. ErrTokenNotFound (an
// errors.Is-matchable wrapper of credstore.ErrNotFound) when unset.
func (s *Store) Token() (*oauth2.Token, error) {
	v, err := s.cs.Get(s.profile, KeyOAuthToken)
	if errors.Is(err, credstore.ErrNotFound) || (err == nil && v == "") {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		// Never embed the value; naming ref/key/op is allowed (§1.12).
		return nil, fmt.Errorf("read %s from %s: %w", KeyOAuthToken, s.ref, err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(v), &tok); err != nil {
		return nil, fmt.Errorf("parse stored token from %s: %w", s.ref, err)
	}
	return &tok, nil
}

// SetToken stores the OAuth token. Ingress-only at init/set-credential, plus
// the single sanctioned non-ingress write: runtime token refresh persisting
// the rotated token back under the active ref (standard §ix / line 174).
func (s *Store) SetToken(tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("serialize token: %w", err)
	}
	if err := s.cs.Set(s.profile, KeyOAuthToken, string(data), credstore.WithOverwrite()); err != nil {
		return fmt.Errorf("store %s at %s: %w", KeyOAuthToken, s.ref, err)
	}
	return nil
}

// DeleteToken removes the token (idempotent: an absent key is not an error —
// §1.7). The Exists pre-check is backend-agnostic: credstore's file backend
// surfaces a raw os "not found" rather than ErrNotFound on Delete.
func (s *Store) DeleteToken() error {
	if ok, _ := s.cs.Exists(s.profile, KeyOAuthToken); !ok {
		return nil
	}
	if err := s.cs.Delete(s.profile, KeyOAuthToken); err != nil && !errors.Is(err, credstore.ErrNotFound) {
		return fmt.Errorf("delete %s at %s: %w", KeyOAuthToken, s.ref, err)
	}
	return nil
}

// HasToken reports presence without returning the value (`config show`,
// `init` overwrite check — §1.6).
func (s *Store) HasToken() bool {
	ok, err := s.cs.Exists(s.profile, KeyOAuthToken)
	return err == nil && ok
}

// Clear removes the whole bundle under the active profile (`config clear`,
// §1.7). Idempotent; scope is the active profile only.
func (s *Store) Clear() ([]string, error) {
	return s.cs.DeleteBundle(s.profile)
}

package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2/google"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
)

// One-time legacy migration (§1.8 / §2.3). Two independent artifacts:
//
//  1. The per-user OAuth token (an access secret): historically in the macOS
//     login Keychain (`security`, service google-readonly / account
//     oauth_token), the Linux Secret Service (`secret-tool`, same coords), or
//     a plaintext token.json fallback. Migrated into the credstore bundle key
//     oauth_token, originals then removed. Conflicts (legacy vs legacy, or
//     legacy vs an existing keyring value) fail loudly per §1.8 — detected
//     before any write/delete; nothing is touched on conflict; no value is
//     ever printed.
//
//  2. The OAuth client JSON (deployment material, §1.2 — NOT a secret):
//     legacy credentials.json relocated to oauth_client_path. Lossless and
//     idempotent; no conflict-fail-loud secret machinery.

// legacyKeychainService is the historical service name (also the secret-tool
// "service" attribute). The account/key was already "oauth_token", so unlike
// slck there is no field rename.
const legacyKeychainService = config.DirName // "google-readonly"

// candidate is one discovered legacy token value.
type candidate struct {
	location string       // non-secret descriptor (never the value)
	value    string       // the serialized oauth2.Token JSON
	deleter  func() error // removes this specific legacy original
}

// migrateLegacyOverwrite runs the one-time migration. overwrite (the §1.8
// `--overwrite` path) forces a legacy value over an existing keyring entry;
// it cannot resolve a legacy-vs-legacy disagreement — the user must pick.
func migrateLegacyOverwrite(s *Store, cfg *config.Config, overwrite bool) error {
	// Deployment material first (independent of the secret; never blocks the
	// token path and never fails loud for a non-secret).
	if err := migrateOAuthClientJSON(cfg); err != nil {
		return err
	}

	cands := discover()
	if len(cands) == 0 {
		return nil // nothing legacy on disk/keychain — the steady state
	}

	plan, err := planMigration(s.service, s.profile, s.ref, cands,
		func() (string, bool) { return currentValue(s) }, overwrite)
	if err != nil {
		return err
	}

	if plan.write != "" {
		var opts []credstore.SetOpt
		if overwrite {
			opts = append(opts, credstore.WithOverwrite())
		}
		if _, err := s.cs.SetBundle(s.profile, map[string]string{KeyOAuthToken: plan.write}, opts...); err != nil {
			return fmt.Errorf("migrate to keyring %s: %w", s.ref, err)
		}
	}

	// Surface the §1.8 signal only for a key actually moved this run. Record
	// the machine-readable block; output.JSON splices it for JSON commands,
	// and root.runRoot flushes the human stderr line otherwise.
	if plan.signal {
		human := fmt.Sprintf("gro: migrated %s into keyring %s (from %s; legacy original removed)",
			plan.change.Field, s.ref, plan.change.From)
		migrationsink.Record(credstore.NewMigrationBlock(plan.change), human)
	}

	for _, del := range plan.cleanups {
		if err := del(); err != nil {
			return fmt.Errorf("migration wrote the keyring but could not remove a legacy original (%s): %w", s.ref, err)
		}
	}
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("migration succeeded but writing config.yml failed: %w", err)
	}
	return nil
}

// migrationPlan is the pure result of resolving discovered candidates.
type migrationPlan struct {
	write    string                    // value to SetBundle under oauth_token ("" = none)
	signal   bool                      // emit the §1.8 signal this run
	change   credstore.MigrationChange // _migration entry / stderr field
	cleanups []func() error            // legacy deleters (post-write)
}

// planMigration is the pure §1.8 resolver: every conflict is detected before
// any mutation is proposed. It performs no I/O — `current` injects the
// existing keyring value lookup — so all branches (legacy-vs-legacy
// disagreement, legacy-vs-target, idempotent equal, overwrite) are
// unit-testable with synthetic candidates.
func planMigration(service, profile, ref string, cands []candidate,
	current func() (string, bool), overwrite bool) (migrationPlan, error) {

	if len(cands) == 0 {
		return migrationPlan{}, nil // nothing legacy — pure no-op
	}

	distinct := map[string]bool{}
	for _, c := range cands {
		distinct[c.value] = true
	}
	target, hasTarget := current()

	switch {
	case len(distinct) > 1:
		// Legacy sources disagree among themselves: --overwrite cannot pick a
		// winner here either (§1.8 — the user must). Always a conflict.
		return migrationPlan{}, conflictErr(service, profile, ref, cands, hasTarget)
	case hasTarget && !overwrite && disagrees(distinct, target):
		return migrationPlan{}, conflictErr(service, profile, ref, cands, hasTarget)
	case hasTarget && !disagrees(distinct, target):
		// Already migrated (values match): no write, just clean up leftover
		// originals from an interrupted prior run. No signal.
		p := migrationPlan{}
		for _, c := range cands {
			p.cleanups = append(p.cleanups, c.deleter)
		}
		return p, nil
	default:
		// Resolvable: one distinct value (or overwrite forcing one).
		p := migrationPlan{
			write:  cands[0].value,
			signal: true,
			change: credstore.MigrationJSONEntry("oauth_token", cands[0].location,
				fmt.Sprintf("keyring:%s/%s/%s", service, profile, KeyOAuthToken)),
		}
		for _, c := range cands {
			p.cleanups = append(p.cleanups, c.deleter)
		}
		return p, nil
	}
}

// currentValue reports the existing credstore value for oauth_token.
func currentValue(s *Store) (string, bool) {
	v, err := s.cs.Get(s.profile, KeyOAuthToken)
	if err != nil || v == "" {
		return "", false
	}
	return v, true
}

func disagrees(distinct map[string]bool, target string) bool {
	for v := range distinct {
		if v != target {
			return true
		}
	}
	return false
}

// conflictErr builds the §1.8 error: names every legacy source and the
// keyring target, never a value (masked or not). Pure — no *Store.
func conflictErr(service, profile, ref string, cands []candidate, hasTarget bool) error {
	locs := make([]string, 0, len(cands)+1)
	for _, c := range cands {
		locs = append(locs, c.location)
	}
	if hasTarget {
		locs = append(locs, fmt.Sprintf("keyring:%s/%s/%s", service, profile, KeyOAuthToken))
	}
	return credstore.MigrationConflictError("gro", "oauth_token", strings.Join(locs, ", "), ref)
}

// Test-only seams: when set, discover() skips the corresponding shell-out so
// the suite is hermetic and never touches the real login Keychain / Secret
// Service. Production never sets them — legacy discovery must run regardless
// of the destination backend (§2.3: a user who opts into keyring.backend:file
// must still have an old `security`/`secret-tool` item migrated).
const (
	legacyKeychainScanDisabledEnv   = "GRO_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN"
	legacySecretToolScanDisabledEnv = "GRO_TEST_DISABLE_LEGACY_SECRETTOOL_SCAN" //nolint:gosec // G101: an env var name, not a credential
)

// discover enumerates every legacy token source that currently exists,
// independent of the destination backend (§2.3). Keychain/secret-tool reads
// are migration-only. The token.json path is the released fallback layout —
// $XDG_CONFIG_HOME/google-readonly/token.json else ~/.config/... — on every
// OS (the legacy code has no %APPDATA% branch).
func discover() []candidate {
	var out []candidate

	if runtime.GOOS == "darwin" && os.Getenv(legacyKeychainScanDisabledEnv) == "" {
		if v, ok := keychainRead(legacyKeychainService, KeyOAuthToken); ok {
			out = append(out, candidate{
				location: fmt.Sprintf("keychain:%s/%s", legacyKeychainService, KeyOAuthToken),
				value:    v,
				deleter:  func() error { return keychainDelete(legacyKeychainService, KeyOAuthToken) },
			})
		}
	}
	if runtime.GOOS == "linux" && os.Getenv(legacySecretToolScanDisabledEnv) == "" {
		if v, ok := secretToolRead(legacyKeychainService, KeyOAuthToken); ok {
			out = append(out, candidate{
				location: fmt.Sprintf("secret-tool:%s/%s", legacyKeychainService, KeyOAuthToken),
				value:    v,
				deleter:  func() error { return secretToolDelete(legacyKeychainService, KeyOAuthToken) },
			})
		}
	}

	if path, err := config.GetTokenPath(); err == nil {
		if v, ok := readLegacyTokenFile(path); ok {
			out = append(out, candidate{
				location: fmt.Sprintf("file:%s", path),
				value:    v,
				deleter:  func() error { return secureDelete(path) },
			})
		}
	}
	return out
}

// readLegacyTokenFile returns the raw token.json contents (a serialized
// oauth2.Token) if the file exists and is non-empty. Missing/empty → no
// candidate (the steady state).
func readLegacyTokenFile(path string) (string, bool) {
	data, err := os.ReadFile(path) //nolint:gosec // path from config dir
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", false
	}
	return v, true
}

func keychainRead(service, account string) (string, bool) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", false
	}
	v := strings.TrimRight(string(out), "\r\n")
	if v == "" {
		return "", false
	}
	return v, true
}

// securityErrItemNotFound is `security`'s exit status when the item is absent
// (errSecItemNotFound). Only this is treated as idempotent success — a
// denial/locked/other failure must surface so we don't silently leave a
// legacy secret behind after "migration".
const securityErrItemNotFound = 44

func keychainDelete(service, account string) error {
	err := exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run()
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == securityErrItemNotFound {
		return nil // already absent — fine for idempotent cleanup
	}
	return fmt.Errorf("remove legacy keychain item %s/%s: %w", service, account, err)
}

func secretToolRead(service, account string) (string, bool) {
	out, err := exec.Command("secret-tool", "lookup",
		"service", service, "account", account).Output()
	if err != nil {
		return "", false
	}
	v := strings.TrimRight(string(out), "\r\n")
	if v == "" {
		return "", false
	}
	return v, true
}

func secretToolDelete(service, account string) error {
	err := exec.Command("secret-tool", "clear",
		"service", service, "account", account).Run()
	if err == nil {
		return nil
	}
	// secret-tool clear is a no-op (exit 0) when nothing matches; a non-zero
	// status is a real failure that must surface.
	return fmt.Errorf("remove legacy secret-tool item %s/%s: %w", service, account, err)
}

// secureDelete overwrites the legacy token file with zeros before removing
// it (defence against forensic recovery), idempotently: an already-gone file
// is success.
func secureDelete(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if f, oerr := os.OpenFile(path, os.O_WRONLY, 0); oerr == nil { //nolint:gosec // G304: path derived from the config dir, not user input
		zeros := make([]byte, info.Size())
		_, _ = f.Write(zeros)
		_ = f.Sync()
		_ = f.Close()
	}
	if rerr := os.Remove(path); rerr != nil && !os.IsNotExist(rerr) {
		return rerr
	}
	return nil
}

// migrateOAuthClientJSON relocates the legacy credentials.json (deployment
// material, §1.2) to cfg.OAuthClientPath. Idempotent by construction: every
// branch that proceeds deletes the legacy file, so a later run sees it absent
// and is silent; the only non-deleting branch (both invalid) is a hard,
// recoverable error the user must resolve, so it cannot loop unnoticed.
func migrateOAuthClientJSON(cfg *config.Config) error {
	legacyPath, err := config.GetCredentialsPath()
	if err != nil {
		return err
	}
	legacyData, lerr := os.ReadFile(legacyPath) //nolint:gosec // path from config dir
	if lerr != nil {
		return nil // legacy absent → nothing to do (steady state)
	}

	target := config.ExpandPath(cfg.OAuthClientPath)
	if target == "" {
		if target, err = config.DefaultOAuthClientPath(); err != nil {
			return err
		}
		cfg.OAuthClientPath = target
	}

	legacyValid := validClientJSON(legacyData)
	targetData, terr := os.ReadFile(target) //nolint:gosec // path from config dir
	targetExists := terr == nil
	targetValid := targetExists && validClientJSON(targetData)

	switch {
	case targetExists && targetValid:
		// A usable client JSON is already installed; the legacy copy is
		// redundant deployment material. Remove it (idempotence).
		if rmErr := os.Remove(legacyPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("remove superseded legacy credentials.json: %w", rmErr)
		}
		fmt.Fprintf(os.Stderr, "Removed superseded legacy credentials.json (OAuth client JSON already present at %s).\n",
			config.ShortenPath(target))
	case !legacyValid && (!targetExists || !targetValid):
		// Both unusable: do not delete anything. Loud, lossless, recoverable.
		return fmt.Errorf(
			"OAuth client JSON migration cannot proceed: legacy %s (sha256:%s) and target %s (%s) are both missing or invalid; install a valid OAuth client JSON at %s",
			config.ShortenPath(legacyPath), fingerprint(legacyData),
			config.ShortenPath(target), targetStateStr(targetExists, targetData),
			config.ShortenPath(target))
	default:
		// Legacy valid, target absent or invalid → install legacy → target.
		if werr := os.WriteFile(target, legacyData, config.OutputFilePerm); werr != nil {
			return fmt.Errorf("write OAuth client JSON %s: %w", config.ShortenPath(target), werr)
		}
		if rmErr := os.Remove(legacyPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("remove legacy credentials.json after copy: %w", rmErr)
		}
		fmt.Fprintf(os.Stderr, "Migrated OAuth client JSON: %s -> %s (deployment material, not a secret).\n",
			config.ShortenPath(legacyPath), config.ShortenPath(target))
	}

	cfg.OAuthClientPath = target
	return nil
}

// validClientJSON reports whether data parses as a Google OAuth client
// config. Scopes do not affect parse validity (they only populate
// Config.Scopes), so none are passed — this also keeps the migration free of
// an internal/auth import (which would create a keychain<->auth cycle).
func validClientJSON(data []byte) bool {
	_, err := google.ConfigFromJSON(data)
	return err == nil
}

func fingerprint(data []byte) string {
	if len(data) == 0 {
		return "absent"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// targetStateStr renders the non-secret state of the target for the
// both-invalid error (presence + fingerprint, never contents).
func targetStateStr(exists bool, data []byte) string {
	if !exists {
		return "absent"
	}
	return "sha256:" + fingerprint(data)
}

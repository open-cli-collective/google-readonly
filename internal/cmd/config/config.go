// Package config implements the gro config command and subcommands.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// configFilesForClear returns the config files `clear --all` should remove,
// deduped by path-identity (Linux collapses old==new). Symmetric with the
// cache side's CacheDirPath + LegacyCacheDir treatment. Both forms
// (config.yml and the legacy config.json) are listed at both the canonical
// and old hand-rolled dirs because LoadConfig still reads new-dir
// config.json when YAML is absent, and the relocation gate can still copy
// either form forward from the old dir. The OAuth client JSON is
// intentionally excluded (deployment material, per the existing trailing
// note in runClear). Resolves WITHOUT creating directories.
func configFilesForClear() ([]string, error) {
	newDir, err := config.GetConfigDirNoCreate()
	if err != nil {
		return nil, err
	}
	oldDir, _ := config.OldHandRolledConfigDir() // best-effort: no HOME → skip legacy

	var paths []string
	seen := map[string]struct{}{}
	add := func(p string) {
		if _, dup := seen[p]; dup {
			return
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	add(filepath.Join(newDir, config.ConfigFileYAML))
	add(filepath.Join(newDir, config.ConfigFile))
	if oldDir != "" {
		add(filepath.Join(oldDir, config.ConfigFileYAML))
		add(filepath.Join(oldDir, config.ConfigFile))
	}
	return paths, nil
}

// configFilesForClearFn is the package-var test seam: tests inject a
// synthetic distinct-old/new path list so Linux CI can exercise the
// macOS/Windows "old != new" branch without OS-specific paths.
var configFilesForClearFn = configFilesForClear

// NewCommand returns the config command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gro configuration",
		Long:  "View and manage gro configuration and authentication status.",
	}
	cmd.AddCommand(newShowCommand())
	cmd.AddCommand(newTestCommand())
	cmd.AddCommand(newClearCommand())
	return cmd
}

func newShowCommand() *cobra.Command {
	var jsonOut, verbose bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display configuration status",
		Long: `Display non-secret configuration status (§1.6): keyring backend,
credential ref, whether the OAuth token is present, and the OAuth client JSON
(deployment material) by path + presence + fingerprint. The token value is
never shown. --verbose inlines the OAuth client JSON contents.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runShow(jsonOut, verbose)
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Emit JSON")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Inline the OAuth client JSON contents")
	return cmd
}

func newTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test Gmail API connectivity",
		Long: `Test the Gmail API connection with the stored token. This is the
installer/runtime smoke check: it runs the same path as a real API command
(including the one-time migration and §1.8 conflict detection).`,
		Args: cobra.NoArgs,
		RunE: runTest,
	}
}

func newClearCommand() *cobra.Command {
	var all, dryRun bool
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove the stored OAuth token (active profile)",
		Long: `Remove the stored OAuth token under the active credential_ref,
forcing re-authentication (§1.7). --all also removes config.yml and the Drive
metadata cache. --dry-run reports what would be removed without removing it.
The OAuth client JSON (deployment material) is never removed.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runClear(all, dryRun)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Also remove config.yml and the Drive metadata cache")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report what would be removed; remove nothing")
	return cmd
}

// showStatus is the §1.6 non-secret view: never the token value, not even a
// masked prefix.
type showStatus struct {
	CredentialRef          string `json:"credential_ref"`
	Backend                string `json:"backend"`
	BackendSource          string `json:"backend_source"`
	PassphraseSource       string `json:"passphrase_source,omitempty"`
	OAuthTokenPresent      bool   `json:"oauth_token_present"`
	OAuthClientPath        string `json:"oauth_client_path"`
	OAuthClientPresent     bool   `json:"oauth_client_present"`
	OAuthClientFingerprint string `json:"oauth_client_fingerprint,omitempty"`
	OAuthClientContents    string `json:"oauth_client_contents,omitempty"`
}

func runShow(jsonOut, verbose bool) error {
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return err
	}

	// OpenNoMigrate: config show is the §1.6 diagnostic and must remain
	// usable during an unresolved §1.8 conflict so the user can see state
	// before remediating (running migration would fail it with a conflict).
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	hasTok, err := st.HasToken()
	if err != nil {
		return err
	}
	backend, src := st.Backend()
	status := showStatus{
		CredentialRef:      st.Ref(),
		Backend:            string(backend),
		BackendSource:      string(src),
		OAuthTokenPresent:  hasTok,
		OAuthClientPath:    config.ShortenPath(cfg.OAuthClientPath),
		OAuthClientPresent: false,
	}
	if backend == credstore.BackendFile {
		status.PassphraseSource = keychain.PassphraseSource(st.Service())
	}
	if data, rerr := os.ReadFile(cfg.OAuthClientPath); rerr == nil { //nolint:gosec // deployment-material path
		status.OAuthClientPresent = true
		status.OAuthClientFingerprint = "sha256:" + fileFingerprint(data)
		if verbose {
			status.OAuthClientContents = redactClientJSON(data)
		}
	}

	if jsonOut {
		return output.JSONStdout(status)
	}

	fmt.Printf("Credential ref:      %s\n", status.CredentialRef)
	fmt.Printf("Backend:             %s (%s)\n", status.Backend, status.BackendSource)
	if status.PassphraseSource != "" {
		fmt.Printf("Passphrase:          %s\n", status.PassphraseSource)
	}
	fmt.Printf("OAuth token:         %s\n", presence(status.OAuthTokenPresent))
	fmt.Printf("OAuth client JSON:   %s\n", status.OAuthClientPath)
	if status.OAuthClientPresent {
		fmt.Printf("  present, %s\n", status.OAuthClientFingerprint)
		if verbose {
			fmt.Printf("  contents:\n%s\n", status.OAuthClientContents)
		}
	} else {
		fmt.Printf("  not found\n")
	}
	if !status.OAuthTokenPresent || !status.OAuthClientPresent {
		fmt.Println()
		fmt.Println("Run 'gro init' to complete setup.")
	}
	return nil
}

func runTest(cmd *cobra.Command, _ []string) error {
	fmt.Println("Testing Gmail API connection...")
	fmt.Println()

	// Open() (NOT OpenNoMigrate): the smoke check must exercise the same path
	// as a real API command, including the one-time migration and §1.8
	// conflict detection — if a conflict blocks real commands it must block
	// this too.
	st, err := keychain.Open()
	if err != nil {
		return err
	}
	has, err := st.HasToken()
	if err != nil {
		_ = st.Close()
		return fmt.Errorf("checking stored token: %w", err)
	}
	if !has {
		_ = st.Close()
		fmt.Println("  OAuth token: Not found")
		fmt.Println()
		fmt.Println("Run 'gro init' to authenticate.")
		return fmt.Errorf("no OAuth token found")
	}
	_ = st.Close()
	fmt.Println("  OAuth token: Found")

	client, err := gmail.NewClient(cmd.Context())
	if err != nil {
		fmt.Println("  Token valid: FAILED")
		fmt.Println()
		fmt.Println("Token may be expired or revoked.")
		fmt.Println("Run 'gro config clear' then 'gro init' to re-authenticate.")
		return fmt.Errorf("creating client: %w", err)
	}
	fmt.Println("  Token valid: OK")

	profile, err := client.GetProfile(cmd.Context())
	if err != nil {
		fmt.Println("  Gmail API:   FAILED")
		return fmt.Errorf("accessing Gmail API: %w", err)
	}
	fmt.Println("  Gmail API:   OK")
	fmt.Printf("  Messages:    %d total\n", profile.MessagesTotal)
	fmt.Println()
	fmt.Printf("Authenticated as: %s\n", profile.EmailAddress)
	return nil
}

func runClear(all, dryRun bool) error {
	// Resolve scrub targets BEFORE opening the keyring (§6.6 pattern 7):
	// `--all` is the user's primary recovery path and must not itself be
	// blocked by the broken state it exists to wipe. Path resolution is
	// side-effect-free; failure here is fatal regardless of --all.
	cfgPaths, err := configFilesForClearFn()
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	// OpenNoMigrate: clear is a §1.8 remediation path ("clear the conflicting
	// entry, then re-run") — running migration first would block it. Under
	// --all an open failure (e.g. invalid keyring.backend in a malformed
	// canonical config) is soft-degraded so the file scrub still runs.
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		if !all {
			return err
		}
		fmt.Fprintf(os.Stderr, "warning: could not open keyring (%v) — proceeding with file scrub only\n", err)
		st = nil
	}
	// Nil-guard the deferred close: a later HasToken soft-degrade may set
	// st = nil after closing it eagerly. Closing nil would panic.
	defer func() {
		if st != nil {
			_ = st.Close()
		}
	}()

	hasTok := false
	if st != nil {
		h, herr := st.HasToken()
		switch {
		case herr == nil:
			hasTok = h
		case all:
			// Symmetric with the OpenNoMigrate soft-degrade: a HasToken
			// failure (e.g. partial backend corruption) under --all must
			// not block the file scrub. Surface a warning, drop the
			// store reference so token branches below skip cleanly.
			fmt.Fprintf(os.Stderr, "warning: could not check stored token (%v) — proceeding with file scrub only\n", herr)
			_ = st.Close()
			st = nil
		default:
			return fmt.Errorf("checking stored token: %w", herr)
		}
	}

	if dryRun {
		switch {
		case st == nil:
			fmt.Println("Would remove: (keyring unavailable — token state unknown)")
		case hasTok:
			fmt.Printf("Would remove: OAuth token at %s\n", st.Ref())
		default:
			fmt.Println("Would remove: (no OAuth token present)")
		}
		if all {
			for _, p := range cfgPaths {
				fmt.Printf("Would remove: %s\n", config.ShortenPath(p))
			}
			// Non-creating resolver: --dry-run must not create or migrate.
			if cacheDir, cerr := config.CacheDirPath(); cerr == nil {
				fmt.Printf("Would remove: Drive metadata cache at %s\n", config.ShortenPath(cacheDir))
			}
		}
		fmt.Println()
		fmt.Println("--dry-run: nothing was removed.")
		return nil
	}

	switch {
	case st == nil:
		// Already warned above; skip token cleanup.
	case hasTok:
		if err := st.DeleteToken(); err != nil {
			return fmt.Errorf("clearing token: %w", err)
		}
		fmt.Printf("Cleared OAuth token from %s.\n", st.Ref())
	default:
		fmt.Println("No OAuth token found to clear.")
	}

	if all {
		for _, p := range cfgPaths {
			switch err := os.Remove(p); {
			case err == nil:
				fmt.Printf("Removed %s.\n", config.ShortenPath(p))
			case os.IsNotExist(err):
				// not present — fine
			default:
				return fmt.Errorf("removing %s: %w", config.ShortenPath(p), err)
			}
		}

		// Drive metadata cache: an explicit full reset removes BOTH the
		// current cache dir and the legacy (pre-B2b) one directly — no
		// cache.New(), so no migrate-then-delete dance and no MkdirAll
		// side-effect. A removal error is surfaced (not silently swallowed):
		// the user must not believe a full reset succeeded if it did not.
		if cacheDir, cerr := config.CacheDirPath(); cerr == nil {
			if rmErr := os.RemoveAll(cacheDir); rmErr != nil {
				return fmt.Errorf("removing Drive metadata cache %s: %w", config.ShortenPath(cacheDir), rmErr)
			}
			fmt.Printf("Removed Drive metadata cache at %s.\n", config.ShortenPath(cacheDir))
		}
		if legacy, lerr := config.LegacyCacheDir(); lerr == nil {
			if rmErr := os.RemoveAll(legacy); rmErr != nil {
				return fmt.Errorf("removing legacy Drive cache %s: %w", config.ShortenPath(legacy), rmErr)
			}
		}
	}

	fmt.Println()
	fmt.Println("Note: the OAuth client JSON (deployment material) is not removed.")
	fmt.Println("Run 'gro init' to re-authenticate.")
	return nil
}

func presence(ok bool) string {
	if ok {
		return "present"
	}
	return "not configured"
}

func fileFingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// redactClientJSON renders the OAuth client JSON for `config show --verbose`
// with every client_secret replaced by "[redacted]". The client JSON is
// deployment material (§1.2), not a per-user access secret, but echoing the
// client_secret onto stdout (CI logs, shell history, log aggregation) is
// needless exposure. If the bytes do not parse, the raw contents are NOT
// inlined — they may still carry the secret.
func redactClientJSON(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return "[OAuth client JSON present but not valid JSON; not inlined]"
	}
	redactSecrets(v)
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "[OAuth client JSON could not be re-encoded; not inlined]"
	}
	return string(out)
}

func redactSecrets(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if k == "client_secret" {
				t[k] = "[redacted]"
				continue
			}
			redactSecrets(val)
		}
	case []any:
		for _, e := range t {
			redactSecrets(e)
		}
	}
}

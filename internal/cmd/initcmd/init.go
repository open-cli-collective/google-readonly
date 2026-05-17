// Package initcmd implements the gro init command — a single-run guided
// OAuth setup wizard built on charmbracelet/huh.
package initcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/huh"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/auth"
	mecmd "github.com/open-cli-collective/google-readonly/internal/cmd/me"
	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/people"
	"github.com/open-cli-collective/google-readonly/internal/view"
)

// initOptions holds command-line options for `gro init`.
type initOptions struct {
	credentialsFile string
	noBrowser       bool
	noVerify        bool
	authCodeStdin   bool
}

// NewCommand returns the init command.
func NewCommand() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set up Google API authentication",
		Long: fmt.Sprintf(`Guided OAuth setup. Walks you through:

  1. Reading your downloaded OAuth client JSON (clipboard, paste, or file path).
  2. Opening the consent URL in your browser.
  3. Pasting the redirect URL back to complete authentication.

After setup, run 'gro me' to see who you're authenticated as.

The wizard first asks how you're getting your credentials.json:
  - Admin-provided (e.g. via 1Password): paste or point to the file.
  - DIY: walks you through creating a Google Cloud project yourself,
    enabling the Gmail API, Google Calendar API, Google Drive API, and
    People API, and downloading OAuth 2.0 Desktop-app credentials.

If you're a Google Workspace admin and want to set up one Internal OAuth app
for your whole org, see:
  %s

You can also copy your credentials.json to the clipboard and run 'gro init' —
it will read, validate, and write it to the config directory for you.`, workspaceAdminsURL),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWith(cmd.Context(), defaultDeps(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.credentialsFile, "credentials-file", "", "Path to a downloaded OAuth client JSON (bypasses interactive paste)")
	cmd.Flags().BoolVar(&opts.noBrowser, "no-browser", false, "Don't try to open the consent URL in a browser")
	cmd.Flags().BoolVar(&opts.noVerify, "no-verify", false, "Skip connectivity verification after setup")
	cmd.Flags().BoolVar(&opts.authCodeStdin, "auth-code-stdin", false, "Read the OAuth authorization code/redirect URL from stdin (two-phase install; implies no browser-open)")

	return cmd
}

// initDeps groups every external collaborator the wizard touches. Tests
// override individual fields; production uses defaultDeps().
type initDeps struct {
	View *view.View

	// FS abstraction so tests can use a temp dir without env shenanigans.
	GetCredentialsPath func() (string, error)
	ReadFile           func(path string) ([]byte, error)
	WriteFile          func(path string, data []byte, perm os.FileMode) error
	Chmod              func(path string, perm os.FileMode) error
	Stat               func(path string) (os.FileInfo, error)

	// Clipboard. Supported is checked first; ReadAll only called if Supported.
	ClipboardSupported func() bool
	ClipboardReadAll   func() (string, error)

	// Browser opener.
	OpenBrowser func(url string) error

	// EnsureMigrated runs/resolves the one-time §1.8 migration up front (a
	// legacy-vs-keyring conflict aborts init loudly).
	EnsureMigrated func() error

	// Token storage (credstore-backed; the keyring is the only store).
	HasStoredToken    func() bool
	SetToken          func(t *oauth2.Token) error
	DeleteToken       func() error
	GetStorageBackend func() string

	// StdinReadAll backs --auth-code-stdin (two-phase install). Test seam.
	StdinReadAll func() (string, error)

	// OAuth.
	ExchangeAuthCode func(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error)
	GetOAuthConfig   func() (*oauth2.Config, error)

	// API verifiers (one Gmail, one People). Both used during init.
	GmailVerify func(ctx context.Context) (string, error) // returns email
	PeopleGetMe func(ctx context.Context) (*people.Profile, error)

	// User-config plumbing.
	LoadConfig    func() (*config.Config, error)
	SaveConfig    func(cfg *config.Config) error
	GetConfigPath func() (string, error)

	// Prompter is the only piece that wraps huh.
	Prompter prompter
}

// prompter abstracts huh's interactive prompts so tests can drive the
// wizard with deterministic inputs.
type prompter interface {
	SelectAudience() (string, error)                          // returns "admin" | "diy"
	SelectCredSource(clipboardSupported bool) (string, error) // returns "clipboard" | "paste" | "file"
	PasteJSON() (string, error)
	FilePath() (string, error)
	ConfirmOpenBrowser() (bool, error)
	PasteRedirectURL() (string, error)
	ConfirmReauth() (bool, error)
	AskCacheTTL(defaultHours int) (int, error)
}

// defaultDeps wires up production collaborators.
func defaultDeps() initDeps {
	return initDeps{
		View: view.New(),
		// The OAuth client JSON is deployment material (§1.2): the wizard
		// writes it to oauth_client_path, not the legacy credentials.json.
		GetCredentialsPath: func() (string, error) {
			cfg, err := config.LoadConfig()
			if err != nil {
				return "", err
			}
			return config.ExpandPath(cfg.OAuthClientPath), nil
		},
		ReadFile:           os.ReadFile,
		WriteFile:          os.WriteFile,
		Chmod:              os.Chmod,
		Stat:               os.Stat,
		ClipboardSupported: func() bool { return !clipboard.Unsupported },
		ClipboardReadAll:   clipboard.ReadAll,
		OpenBrowser:        browser.OpenURL,
		EnsureMigrated:     ensureMigrated,
		HasStoredToken:     storeHasToken,
		SetToken:           storeSetToken,
		DeleteToken:        storeDeleteToken,
		GetStorageBackend:  storeBackendLabel,
		StdinReadAll:       readAllStdin,
		ExchangeAuthCode:   auth.ExchangeAuthCode,
		GetOAuthConfig:     auth.GetOAuthConfig,
		GmailVerify: func(ctx context.Context) (string, error) {
			c, err := gmail.NewClient(ctx)
			if err != nil {
				return "", err
			}
			p, err := c.GetProfile(ctx)
			if err != nil {
				return "", err
			}
			return p.EmailAddress, nil
		},
		PeopleGetMe: func(ctx context.Context) (*people.Profile, error) {
			c, err := people.NewClient(ctx)
			if err != nil {
				return nil, err
			}
			return c.GetMe(ctx)
		},
		LoadConfig:    config.LoadConfig,
		SaveConfig:    config.SaveConfig,
		GetConfigPath: config.GetConfigPath,
		Prompter:      huhPrompter{},
	}
}

// ensureMigrated runs (and resolves) the one-time §1.8 migration up front via
// the full keychain.Open() path, surfacing a legacy-vs-keyring conflict as a
// hard error instead of swallowing it. Closes a self-inflicted conflict: if
// init left a legacy original (token.json / old security / secret-tool) in
// place and then wrote a fresh OAuth token to the keyring, the next real
// command's Open() would hit the §1.8 conflict. After this runs, legacy
// originals are gone (or init aborted loudly), so the storeSetToken /
// storeHasToken / storeDeleteToken paths can stay non-migrating. A genuine
// fresh install migrates nothing (no-op).
func ensureMigrated() error { return keychain.EnsureMigrated() }

// storeHasToken backs the wizard's "is a token already present?" gate.
// Non-migrating: ensureMigrated already ran the one-time migration up front.
// Fail closed: a keyring error returns true (assume a token may be present)
// so the wizard shows its overwrite confirmation rather than silently
// skipping it and clobbering a token that might actually be there. The
// keyring is then re-exercised by storeSetToken, which surfaces the error.
func storeHasToken() bool {
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return true
	}
	defer func() { _ = st.Close() }()
	has, herr := st.HasToken()
	if herr != nil {
		return true
	}
	return has
}

func storeSetToken(t *oauth2.Token) error {
	// OpenNoMigrate (not OpenRef("")): symmetric with storeHasToken /
	// storeDeleteToken — same configured ref, non-migrating (ensureMigrated
	// already ran the one-time migration up front).
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	return st.SetToken(t)
}

func storeDeleteToken() error {
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	return st.DeleteToken()
}

func storeBackendLabel() string {
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return "keyring"
	}
	defer func() { _ = st.Close() }()
	b, _ := st.Backend()
	if b == "" {
		return "keyring"
	}
	return string(b)
}

func readAllStdin() (string, error) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	// Trim: a piped auth code carries a trailing newline that would
	// otherwise be sent verbatim to ExchangeAuthCode (opaque OAuth error).
	// Mirrors setcred.readValue.
	return strings.TrimSpace(string(b)), nil
}

// runWith is the testable entry point for the wizard. NewCommand wraps it.
func runWith(ctx context.Context, d initDeps, opts *initOptions) error {
	// Step 0: run/resolve the one-time §1.8 migration up front. A legacy
	// token is migrated into the keyring (its original removed) before the
	// wizard decides anything; a legacy-vs-keyring conflict aborts loudly
	// rather than letting init write a fresh token alongside a stale legacy
	// original that the next command would conflict on.
	if d.EnsureMigrated != nil {
		if err := d.EnsureMigrated(); err != nil {
			return fmt.Errorf("resolving credential migration: %w", err)
		}
	}

	credPath, err := d.GetCredentialsPath()
	if err != nil {
		return fmt.Errorf("getting credentials path: %w", err)
	}

	// Step 1: ensure credentials.json exists.
	if err := ensureCredentials(d, opts, credPath); err != nil {
		return err
	}

	// Step 2: snapshot config.json existence BEFORE we save granted scopes.
	// We use this snapshot to gate the cache-TTL prompt: if config existed
	// already, we don't re-ask. Today's code samples this AFTER scope save,
	// which made the prompt unreachable on first runs.
	configPath, err := d.GetConfigPath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}
	configExistedBefore := false
	if _, err := d.Stat(configPath); err == nil {
		configExistedBefore = true
	}

	// Step 3: token resolution.
	handled, err := tryExistingToken(ctx, d, opts, configExistedBefore)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	// Step 4: OAuth flow.
	oauthCfg, err := d.GetOAuthConfig()
	if err != nil {
		return fmt.Errorf("loading OAuth config: %w", err)
	}

	authURL := auth.GetAuthURL(oauthCfg)
	if !opts.authCodeStdin && !opts.noBrowser {
		open, err := d.Prompter.ConfirmOpenBrowser()
		if err != nil {
			return err
		}
		if open {
			if err := d.OpenBrowser(authURL); err != nil {
				d.View.Info("Could not open browser automatically (%v).", err)
			}
		}
	}
	d.View.Println("If your browser didn't open, paste this URL into it:")
	d.View.Println("")
	d.View.Println("  " + authURL)
	d.View.Println("")

	// Two-phase install: --auth-code-stdin reads the code/redirect URL from
	// stdin (the installer pauses between "open URL" and "feed code back")
	// instead of the interactive prompt. The value is never echoed.
	var codeInput string
	if opts.authCodeStdin {
		codeInput, err = d.StdinReadAll()
	} else {
		codeInput, err = d.Prompter.PasteRedirectURL()
	}
	if err != nil {
		return err
	}
	code := extractAuthCode(codeInput)
	if code == "" {
		return errors.New("no authorization code found in input")
	}

	token, err := d.ExchangeAuthCode(ctx, oauthCfg, code)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}
	if err := d.SetToken(token); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	d.View.Success("Token saved to %s", d.GetStorageBackend())

	// Step 5: persist granted scopes (creates config.json if missing).
	cfg, cfgErr := d.LoadConfig()
	if cfgErr != nil {
		cfg = &config.Config{CacheTTLHours: config.DefaultCacheTTLHours}
	}
	cfg.GrantedScopes = auth.AllScopes
	if saveErr := d.SaveConfig(cfg); saveErr != nil {
		d.View.Error("Warning: saving granted scopes: %v", saveErr)
	}

	// Step 6: cache TTL prompt (only on first-run, per the snapshot).
	if !configExistedBefore {
		ttl, err := d.Prompter.AskCacheTTL(config.DefaultCacheTTLHours)
		if err != nil {
			return err
		}
		cfg.CacheTTLHours = ttl
		if saveErr := d.SaveConfig(cfg); saveErr != nil {
			d.View.Error("Warning: saving cache TTL: %v", saveErr)
		}
	}

	// Step 7: verify + render `gro me` one-liner. People failure is fatal —
	// init's success contract is that you can immediately run `gro me`.
	if !opts.noVerify {
		email, err := d.GmailVerify(ctx)
		if err != nil {
			return fmt.Errorf("verifying Gmail API: %w", err)
		}
		d.View.Success("Verified Gmail API for %s", email)

		profile, err := d.PeopleGetMe(ctx)
		if err != nil {
			return fmt.Errorf("verifying People API (enable it at https://console.cloud.google.com if not already): %w", err)
		}
		d.View.Println("")
		mecmd.RenderOneLiner(d.View.Out, profile)
	}

	d.View.Println("")
	d.View.Println("Setup complete! Try:")
	d.View.Println("  gro me")
	d.View.Println("  gro mail search \"is:unread\"")
	return nil
}

// tryExistingToken handles the case where a token is already stored.
// Returns (handled=true, nil) if init is done; (handled=false, nil) if the
// caller should fall through to the OAuth flow; (_, err) on errors.
//
// This is the load-bearing piece for #107's stale-scope fix: a Gmail-valid
// but People-insufficient token (typical of users who upgraded gro) must
// trigger re-auth here, otherwise `gro me`'s "run gro init" message
// produces an infinite remediation loop.
func tryExistingToken(ctx context.Context, d initDeps, opts *initOptions, configExistedBefore bool) (bool, error) {
	if !d.HasStoredToken() {
		return false, nil
	}

	// Loud-and-early stale-scope check from the recorded scopes. This runs
	// regardless of --no-verify because letting --no-verify skip it would
	// re-open the same remediation loop #107 is trying to close.
	if cfg, err := d.LoadConfig(); err == nil {
		if msg := auth.CheckScopesMigration(cfg.GrantedScopes); msg != "" {
			d.View.Error("Recorded scopes are stale.")
			d.View.Println(msg)
			if err := promptAndDeleteForReauth(d); err != nil {
				return false, err
			}
			return false, nil
		}
	}

	// --no-verify keeps the historical "accept token, skip API calls" semantic
	// for everything past the recorded-scope check above.
	if opts.noVerify {
		d.View.Success("Existing token accepted (--no-verify; API not called)")
		return true, finishExisting(d, configExistedBefore, nil /* no profile */)
	}

	email, err := d.GmailVerify(ctx)
	if err != nil {
		if isAuthError(err) {
			d.View.Error("Stored token is expired or revoked.")
			if err := promptAndDeleteForReauth(d); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}
	d.View.Success("Already authenticated as %s", email)

	// People verify catches scope-stale tokens that Gmail accepts. Reuse the
	// returned profile for the success-line render so we don't make a second
	// API call.
	profile, err := d.PeopleGetMe(ctx)
	if err != nil {
		if people.IsInsufficientScopeError(err) {
			d.View.Error("Token is missing People API scope.")
			if err := promptAndDeleteForReauth(d); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, fmt.Errorf("verifying People API: %w", err)
	}

	return true, finishExisting(d, configExistedBefore, profile)
}

// promptAndDeleteForReauth asks the user whether to re-auth and clears the
// stored token if they confirm. Returns nil (caller should fall through to
// the OAuth flow) on confirm; returns an error on decline.
func promptAndDeleteForReauth(d initDeps) error {
	confirm, err := d.Prompter.ConfirmReauth()
	if err != nil {
		return err
	}
	if !confirm {
		d.View.Info("Run 'gro config clear' to remove the stored token.")
		return errors.New("re-authentication declined")
	}
	if err := d.DeleteToken(); err != nil {
		return fmt.Errorf("clearing token: %w", err)
	}
	return nil
}

// finishExisting handles the post-success path when a token was already
// present and verified. profile is the People profile fetched during verify
// (may be nil for the --no-verify path); when present we render the gro me
// one-liner without a second API call.
//
// We deliberately do NOT touch GrantedScopes here: the existing token's
// granted scopes are an unknown to this run, and writing auth.AllScopes
// would falsely claim the token is current. Only a fresh OAuth flow knows
// what was just granted.
func finishExisting(d initDeps, configExistedBefore bool, profile *people.Profile) error {
	if profile != nil {
		d.View.Println("")
		mecmd.RenderOneLiner(d.View.Out, profile)
	}

	if !configExistedBefore {
		// Edge case: token in keychain but no config file. Ask for TTL only.
		ttl, err := d.Prompter.AskCacheTTL(config.DefaultCacheTTLHours)
		if err != nil {
			return err
		}
		cfg, cfgErr := d.LoadConfig()
		if cfgErr != nil {
			cfg = &config.Config{}
		}
		cfg.CacheTTLHours = ttl
		if saveErr := d.SaveConfig(cfg); saveErr != nil {
			d.View.Error("Warning: saving config: %v", saveErr)
		}
	}
	return nil
}

// ensureCredentials makes sure credentials.json exists at credPath, populating
// it from --credentials-file or the interactive wizard if needed.
func ensureCredentials(d initDeps, opts *initOptions, credPath string) error {
	// --credentials-file flag wins.
	if opts.credentialsFile != "" {
		expanded, err := expandTilde(opts.credentialsFile)
		if err != nil {
			return err
		}
		return importFromFile(d, expanded, credPath)
	}

	if _, err := d.Stat(credPath); err == nil {
		return nil
	}

	// Otherwise drive the wizard. Ask once whether the user is admin-provisioned
	// or doing their own Google Cloud setup, so we only show the steps they need.
	audience, err := d.Prompter.SelectAudience()
	if err != nil {
		return err
	}

	d.View.Println("")
	switch audience {
	case "admin":
		d.View.Println("Your admin should have shared credentials.json (e.g. via 1Password).")
		d.View.Println("Paste the JSON, or point to the file when prompted.")
	case "diy":
		d.View.Println("Set up Google OAuth credentials at https://console.cloud.google.com:")
		d.View.Println("  1. Create or select a project.")
		d.View.Println("  2. Enable APIs: Gmail, Google Calendar, Google Drive, and People")
		d.View.Println("     (the People API powers both 'gro contacts' and 'gro me').")
		d.View.Println("  3. Create OAuth 2.0 Desktop-app credentials.")
		d.View.Println("  4. Copy the JSON to your clipboard, OR download the JSON file.")
		d.View.Println("")
		d.View.Println("Optional: publish your OAuth app to avoid 7-day token expiry.")
		d.View.Println("")
		d.View.Println("Workspace admin? Set up an Internal OAuth app once for your whole org:")
		d.View.Println("  " + workspaceAdminsURL)
	default:
		return fmt.Errorf("unknown audience: %s", audience)
	}
	d.View.Println("")

	// Up to 3 attempts to recover from *content* errors (unreadable
	// clipboard, garbage JSON, missing file). User-aborts from the prompter
	// (huh.ErrUserAborted via Ctrl-C) propagate immediately, as they should.
	for attempt := 0; attempt < 3; attempt++ {
		choice, err := d.Prompter.SelectCredSource(d.ClipboardSupported())
		if err != nil {
			return err
		}

		var blob []byte
		switch choice {
		case "clipboard":
			s, err := d.ClipboardReadAll()
			if err != nil {
				d.View.Error("Reading clipboard: %v", err)
				continue
			}
			blob = []byte(s)
		case "paste":
			s, err := d.Prompter.PasteJSON()
			if err != nil {
				return err
			}
			blob = []byte(s)
		case "file":
			pathInput, err := d.Prompter.FilePath()
			if err != nil {
				return err
			}
			expanded, err := expandTilde(pathInput)
			if err != nil {
				d.View.Error("%v", err)
				continue
			}
			blob, err = d.ReadFile(expanded)
			if err != nil {
				d.View.Error("Reading %s: %v", expanded, err)
				continue
			}
		default:
			return fmt.Errorf("unknown choice: %s", choice)
		}

		if err := writeCredentials(d, credPath, blob); err != nil {
			d.View.Error("%v", err)
			continue
		}
		d.View.Success("Credentials saved to %s", credPath)
		return nil
	}
	return errors.New("could not obtain valid credentials.json after 3 attempts")
}

// importFromFile reads, validates, and writes credentials.json from a path.
func importFromFile(d initDeps, srcPath, dstPath string) error {
	blob, err := d.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcPath, err)
	}
	return writeCredentials(d, dstPath, blob)
}

// writeCredentials validates blob as OAuth client JSON and writes it to dst at
// 0600. We chmod after WriteFile because os.WriteFile won't tighten an
// already-existing file's permissions.
func writeCredentials(d initDeps, dst string, blob []byte) error {
	if _, err := google.ConfigFromJSON(blob, auth.AllScopes...); err != nil {
		return fmt.Errorf("invalid OAuth client JSON: %w", err)
	}
	// Ensure parent dir exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	if err := d.WriteFile(dst, blob, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	if err := d.Chmod(dst, 0600); err != nil {
		return fmt.Errorf("chmod %s: %w", dst, err)
	}
	return nil
}

func expandTilde(p string) (string, error) {
	if !strings.HasPrefix(p, "~") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if p == "~" {
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// extractAuthCode is preserved from the original implementation.
func extractAuthCode(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://localhost") || strings.HasPrefix(input, "https://localhost") {
		if u, err := url.Parse(input); err == nil {
			return u.Query().Get("code")
		}
		return ""
	}
	return input
}

// isAuthError is preserved from the original implementation.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *googleapi.Error
	if ok := errorAs(err, &apiErr); ok {
		return apiErr.Code == http.StatusUnauthorized
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") &&
		(strings.Contains(errStr, "Invalid Credentials") ||
			strings.Contains(errStr, "invalid_grant") ||
			strings.Contains(errStr, "Token has been expired or revoked"))
}

var errorAs = errors.As

// workspaceAdminsURL points to the repo's Workspace-admin walkthrough.
// Referenced from both cmd.Long and the runtime wizard, so installed-CLI
// users (Homebrew/Chocolatey/Winget) reach it without a local checkout.
const workspaceAdminsURL = "https://github.com/open-cli-collective/google-readonly/blob/main/WORKSPACE_ADMINS.md"

// huhPrompter is the production prompter — wraps huh.
type huhPrompter struct{}

func (huhPrompter) SelectAudience() (string, error) {
	var choice string
	err := huh.NewSelect[string]().
		Title("How are you getting your OAuth credentials?").
		Options(
			huh.NewOption("My admin gave me a credentials.json (e.g. via 1Password)", "admin"),
			huh.NewOption("I'll set up my own Google Cloud project", "diy"),
		).
		Value(&choice).
		Run()
	return choice, err
}

func (huhPrompter) SelectCredSource(clipboardSupported bool) (string, error) {
	options := []huh.Option[string]{}
	if clipboardSupported {
		options = append(options, huh.NewOption("Read JSON from clipboard", "clipboard"))
	}
	options = append(options,
		huh.NewOption("Paste JSON in terminal", "paste"),
		huh.NewOption("Point to a file path", "file"),
	)
	var choice string
	err := huh.NewSelect[string]().
		Title("How would you like to provide credentials.json?").
		Options(options...).
		Value(&choice).
		Run()
	return choice, err
}

func (huhPrompter) PasteJSON() (string, error) {
	var s string
	err := huh.NewText().
		Title("Paste credentials.json contents").
		Description("Press Tab when done; Ctrl+C to cancel.").
		Value(&s).
		Validate(validateOAuthJSON).
		Run()
	return s, err
}

func (huhPrompter) FilePath() (string, error) {
	var s string
	err := huh.NewInput().
		Title("Path to credentials.json").
		Placeholder("~/Downloads/credentials.json").
		Value(&s).
		Validate(func(in string) error {
			if strings.TrimSpace(in) == "" {
				return errors.New("path is required")
			}
			return nil
		}).
		Run()
	return s, err
}

func (huhPrompter) ConfirmOpenBrowser() (bool, error) {
	var open bool
	err := huh.NewConfirm().
		Title("Open the consent URL in your browser?").
		Description("If you say no, the URL will be printed for you to copy manually.").
		Affirmative("Open").
		Negative("Just print").
		Value(&open).
		Run()
	return open, err
}

func (huhPrompter) PasteRedirectURL() (string, error) {
	var s string
	err := huh.NewInput().
		Title("Paste the redirect URL or the code value").
		Description("After clicking 'Allow', your browser will redirect to a localhost URL that shows an error — that's expected. Copy the entire URL from the address bar and paste it here.").
		Value(&s).
		Validate(func(in string) error {
			if extractAuthCode(in) == "" {
				return errors.New("no authorization code found")
			}
			return nil
		}).
		Run()
	return s, err
}

func (huhPrompter) ConfirmReauth() (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title("Re-authenticate now?").
		Description("This will clear the existing token and start a fresh OAuth flow.").
		Affirmative("Re-auth").
		Negative("Cancel").
		Value(&ok).
		Run()
	return ok, err
}

func (huhPrompter) AskCacheTTL(defaultHours int) (int, error) {
	var s string
	err := huh.NewInput().
		Title("Cache TTL for Drive metadata (hours)").
		Placeholder(strconv.Itoa(defaultHours)).
		Value(&s).
		Validate(func(in string) error {
			in = strings.TrimSpace(in)
			if in == "" {
				return nil
			}
			n, err := strconv.Atoi(in)
			if err != nil || n <= 0 {
				return errors.New("must be a positive integer")
			}
			return nil
		}).
		Run()
	if err != nil {
		return 0, err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultHours, nil
	}
	return strconv.Atoi(s)
}

// validateOAuthJSON is exposed so tests can poke it directly.
func validateOAuthJSON(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("empty")
	}
	if _, err := google.ConfigFromJSON([]byte(s), auth.AllScopes...); err != nil {
		return fmt.Errorf("invalid OAuth client JSON: %w", err)
	}
	return nil
}

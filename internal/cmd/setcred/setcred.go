// Package setcred implements `gro set-credential` — the low-level,
// scriptable credential ingress (§1.5.2). It accepts the OAuth token only
// via stdin or a named env var, never as a flag/positional value, and only
// for the allowed key. This lets an installer pre-seed a token without the
// interactive browser dance.
package setcred

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/keychain"
)

type options struct {
	ref     string
	key     string
	stdin   bool
	fromEnv string
	in      io.Reader // test seam
}

// NewCmd builds the `gro set-credential` command.
func NewCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "set-credential --key oauth_token (--stdin | --from-env NAME)",
		Short: "Store the OAuth token in the keyring (stdin or env ingress)",
		Long: `Store the per-user OAuth token in the OS keyring (§1.5.2).

The value (a serialized oauth2.Token JSON object) is read ONLY from stdin
(--stdin) or a named environment variable (--from-env NAME) — never from a
flag or positional argument. Only the key 'oauth_token' is accepted.

  op read 'op://Vault/gro/oauth_token' | gro set-credential --key oauth_token --stdin
  gro set-credential --key oauth_token --from-env GRO_OAUTH_TOKEN`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if opts.in == nil {
				opts.in = c.InOrStdin()
			}
			return run(opts)
		},
	}
	cmd.Flags().StringVar(&opts.ref, "ref", "", "Credential ref (default: config.yml credential_ref)")
	cmd.Flags().StringVar(&opts.key, "key", "", "Key to set: oauth_token")
	cmd.Flags().BoolVar(&opts.stdin, "stdin", false, "Read the token from stdin")
	cmd.Flags().StringVar(&opts.fromEnv, "from-env", "", "Read the token from this env var")
	return cmd
}

func run(opts *options) error {
	if opts.key == "" {
		return fmt.Errorf("--key is required (oauth_token)")
	}
	if opts.key != keychain.KeyOAuthToken {
		return fmt.Errorf("key %q not allowed (allowed: %s)", opts.key, keychain.KeyOAuthToken)
	}
	if opts.stdin == (opts.fromEnv != "") {
		return fmt.Errorf("exactly one of --stdin or --from-env is required")
	}

	value, err := readValue(opts)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("empty secret value rejected")
	}

	// Validate it is a usable oauth2.Token; never echo the value (§1.12).
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(value), &tok); err != nil {
		return fmt.Errorf("value is not a valid oauth2.Token JSON object: %w", err)
	}
	if tok.AccessToken == "" && tok.RefreshToken == "" {
		return fmt.Errorf("token has neither an access nor a refresh token")
	}

	// §1.8: when targeting the default ref, run the one-time legacy migration
	// first (shared keychain.EnsureMigrated, same guarantee as init).
	// Otherwise a pre-existing legacy token.json + this fresh keyring write
	// would collide on the next real command's Open() with a §1.8 conflict. A
	// genuine conflict here aborts loudly (the user must resolve it, not
	// silently overwrite via this scriptable path). An explicit --ref never
	// migrates — the one-time migration only ever targets the canonical
	// configured ref (see keychain.OpenRef).
	migrated := false
	if opts.ref == "" {
		if merr := keychain.EnsureMigrated(); merr != nil {
			return merr
		}
		migrated = true
	}

	st, err := keychain.OpenRef(opts.ref) // ingress: runMigration=false
	if err != nil {
		if migrated {
			// The legacy original may already have been consumed by the
			// migration above; make the failure actionable.
			return fmt.Errorf("legacy migration succeeded but the keyring write could not be opened (run 'gro init' to re-authenticate): %w", err)
		}
		return err
	}
	defer func() { _ = st.Close() }()

	if err := st.SetToken(&tok); err != nil {
		return err
	}
	// Naming the key/ref is fine; the value is never printed (§1.12).
	fmt.Printf("Stored %s in %s\n", opts.key, st.Ref())
	return nil
}

func readValue(opts *options) (string, error) {
	if opts.fromEnv != "" {
		// TrimSpace mirrors the --stdin path: a trailing newline from
		// `export X=$(cat token.json)` would otherwise make the JSON parse
		// fail with an opaque error.
		v := strings.TrimSpace(os.Getenv(opts.fromEnv))
		if v == "" {
			// Name the variable (never the value) so the user knows what to
			// populate (§1.12: the var name is not secret).
			return "", fmt.Errorf("--from-env %s is empty or unset", opts.fromEnv)
		}
		return v, nil
	}
	r := opts.in
	if r == nil {
		r = os.Stdin
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read token from stdin: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

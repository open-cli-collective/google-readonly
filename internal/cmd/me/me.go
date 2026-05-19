// Package me implements the `gro me` command — a token-dense one-liner
// describing the currently authenticated Google user, modeled on `jtk me`.
package me

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"

	"github.com/open-cli-collective/google-readonly/internal/auth"
	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/people"
)

// errReauth is the well-known error returned when the user must re-run `gro init`.
var errReauth = errors.New("re-authentication required")

// NewCommand returns the `me` cobra command.
func NewCommand() *cobra.Command {
	var (
		idOnly     bool
		extended   bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show the currently authenticated Google user",
		Long: `Show the currently authenticated Google user as a token-dense one-liner:
resourceName | displayName | primaryEmail.

Empty fields render as "-"; embedded pipes are escaped to "\|".

Data comes from the People API people/me endpoint.`,
		Example: `  # One-liner (resourceName | displayName | primaryEmail)
  gro me

  # Just the primary email (for scripting)
  gro me --id

  # Add granted scopes, token expiry, and storage backend
  gro me --extended

  # JSON output
  gro me --json`,
		Args: cobra.NoArgs,
		// SilenceErrors so errReauth's actionable message (already written to
		// stderr by run()) isn't shadowed by cobra's "Error: re-authentication
		// required" prefix line.
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), idOnly, extended, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&idOnly, "id", false, "Print only the primary email")
	cmd.Flags().BoolVar(&extended, "extended", false, "Add granted scopes, token expiry, and storage backend")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Emit JSON")
	cmd.MarkFlagsMutuallyExclusive("id", "extended")

	return cmd
}

func run(ctx context.Context, out, errOut io.Writer, idOnly, extended, jsonOutput bool) error {
	// Loud-and-early stale-scope check (only fires when scopes were recorded).
	if cfg, err := config.LoadConfigForRuntime(); err == nil {
		if msg := auth.CheckScopesMigration(cfg.GrantedScopes); msg != "" {
			_, _ = fmt.Fprintln(errOut, msg)
			return errReauth
		}
	}

	client, err := ClientFactory(ctx)
	if err != nil {
		return fmt.Errorf("creating People client: %w", err)
	}

	profile, err := client.GetMe(ctx)
	if err != nil {
		// Insufficient-scope 403s mean the token is too narrow for the People
		// API even if granted_scopes claims otherwise. Other 403s pass through.
		if people.IsInsufficientScopeError(err) {
			_, _ = fmt.Fprintln(errOut, "Insufficient OAuth scopes for People API.\nRun 'gro init' to re-authenticate with the updated scopes.")
			return errReauth
		}
		// Detect API-not-enabled separately to give a clearer hint.
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 403 {
			return fmt.Errorf("people API request failed (HTTP 403): %s\n\nIf the People API is disabled, enable it at https://console.cloud.google.com and retry", apiErr.Message)
		}
		return fmt.Errorf("getting current user: %w", err)
	}

	// Read token expiry AFTER the API call so any refresh by
	// PersistentTokenSource is reflected.
	var extras Extras
	if extended {
		extras = gatherExtras()
		extras.GrantedScopes = grantedScopes()
	}

	if jsonOutput {
		return RenderJSON(out, profile, extras, idOnly, extended)
	}

	switch {
	case idOnly:
		RenderID(out, profile)
	case extended:
		RenderExtended(out, profile, extras)
	default:
		RenderOneLiner(out, profile)
	}
	return nil
}

// grantedScopes returns the scopes recorded in config. If no record exists
// (no config file, or empty list) we return nil — claiming auth.AllScopes
// would overstate what an older token actually consented to.
func grantedScopes() []string {
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return nil
	}
	return cfg.GrantedScopes
}

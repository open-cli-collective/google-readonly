// Package root provides the top-level gro command and global flags.
package root

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-readonly/internal/cmd/calendar"
	"github.com/open-cli-collective/google-readonly/internal/cmd/config"
	"github.com/open-cli-collective/google-readonly/internal/cmd/contacts"
	"github.com/open-cli-collective/google-readonly/internal/cmd/drive"
	"github.com/open-cli-collective/google-readonly/internal/cmd/initcmd"
	"github.com/open-cli-collective/google-readonly/internal/cmd/mail"
	"github.com/open-cli-collective/google-readonly/internal/cmd/me"
	"github.com/open-cli-collective/google-readonly/internal/cmd/refreshcmd"
	"github.com/open-cli-collective/google-readonly/internal/cmd/setcred"
	"github.com/open-cli-collective/google-readonly/internal/keychain"
	"github.com/open-cli-collective/google-readonly/internal/log"
	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
	"github.com/open-cli-collective/google-readonly/internal/version"
)

var (
	verbose bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "gro",
	Short: "A non-destructive CLI for Google services",
	Long: `gro is a non-destructive command-line interface for Google services.

It provides commands for reading and organizing Gmail messages, Google Calendar
events, Google Contacts, and Google Drive files. Organizational operations
include labeling, archiving, starring, RSVP, and group management. No send,
delete, or trash operations are possible.

To get started, run:
  gro init

This will guide you through OAuth setup for Google API access.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		log.Verbose = verbose
		if noColor {
			lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
		}
		return WireBackendSelection(cmd)
	},
}

// WireBackendSelection validates the user-supplied --backend flag and
// records it for the next keychain.Open* call. Cobra-layer only — it
// does NOT load config; openWith binds the flag pair against
// cfg.Keyring.Backend at the single credstore.Open call site.
//
// Exported so a subcommand that defines its own PersistentPreRunE can
// call it explicitly: cobra does NOT chain PersistentPreRunE, so a
// shadowing subcommand silently loses the wiring without this hook.
// gro has no shadowers today; the regression test guards the pattern.
//
// Reads via cmd.Flag() so persistent-flag inheritance works from any
// subcommand path.
func WireBackendSelection(cmd *cobra.Command) error {
	var value string
	var changed bool
	if bf := cmd.Flag(cccredstore.BackendFlagName); bf != nil {
		value = bf.Value.String()
		changed = bf.Changed
	}
	if err := cccredstore.BindBackendFlag(&cccredstore.Options{}, value, changed, ""); err != nil {
		return fmt.Errorf("--%s: %w", cccredstore.BackendFlagName, err)
	}
	keychain.SetBackendFlagOverride(value, changed)
	return nil
}

// Execute runs the root command with a background context
func Execute() {
	ExecuteContext(context.Background())
}

// ExecuteContext runs the root command with the given context. os.Exit stays
// strictly AFTER runRoot returns so runRoot's deferred FlushMigrationNotice
// is never skipped by the exit (it would be if the defer lived here).
func ExecuteContext(ctx context.Context) {
	if err := runRoot(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runRoot owns the deferred §1.8 migration-notice flush. The defer fires on
// success AND error, before any os.Exit — the reliable "finally" hook
// (Cobra's PersistentPostRunE is skipped on a RunE error, which would lose
// the signal if the one-time migration succeeded but the command then
// failed). A JSON command consumes the record via output.JSON, so this is a
// no-op for it; everything else gets the human stderr line. Stderr never
// corrupts a --json stdout body.
func runRoot(ctx context.Context) error {
	defer migrationsink.FlushMigrationNotice(os.Stderr)
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Set custom version template to include commit and build date
	rootCmd.SetVersionTemplate("gro " + version.Info() + "\n")

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output for debugging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().String(cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())

	// Register commands
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(setcred.NewCmd())
	rootCmd.AddCommand(me.NewCommand())
	rootCmd.AddCommand(mail.NewCommand())
	rootCmd.AddCommand(calendar.NewCommand())
	rootCmd.AddCommand(contacts.NewCommand())
	rootCmd.AddCommand(drive.NewCommand())
	rootCmd.AddCommand(refreshcmd.NewCommand())
}

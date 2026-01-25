package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/cmd/calendar"
	"github.com/open-cli-collective/google-readonly/internal/cmd/config"
	"github.com/open-cli-collective/google-readonly/internal/cmd/contacts"
	"github.com/open-cli-collective/google-readonly/internal/cmd/drive"
	"github.com/open-cli-collective/google-readonly/internal/cmd/initcmd"
	"github.com/open-cli-collective/google-readonly/internal/cmd/mail"
	"github.com/open-cli-collective/google-readonly/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "gro",
	Short: "A read-only CLI for Google services",
	Long: `gro is a command-line interface for read-only access to Google services.

It provides commands for reading Gmail messages, Google Calendar events,
and Google Drive files without modifying any data.

To get started, run:
  gro init

This will guide you through OAuth setup for Google API access.`,
	Version: version.Version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Set custom version template to include commit and build date
	rootCmd.SetVersionTemplate("gro " + version.Info() + "\n")

	// Register commands
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(mail.NewCommand())
	rootCmd.AddCommand(calendar.NewCommand())
	rootCmd.AddCommand(contacts.NewCommand())
	rootCmd.AddCommand(drive.NewCommand())
}

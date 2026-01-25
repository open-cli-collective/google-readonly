package drive

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the drive parent command with subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "drive",
		Aliases: []string{"files"},
		Short:   "Google Drive commands",
		Long: `Read-only access to Google Drive files and folders.

This command group provides Google Drive functionality:
- list: List files in Drive or a specific folder
- search: Search for files by name, content, type, or date
- get: Get detailed metadata for a file
- download: Download files or export Google Docs
- tree: Display folder structure

Examples:
  gro drive list
  gro drive search "quarterly report"
  gro drive get <file-id>
  gro drive download <file-id> --format pdf`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newSearchCommand())

	return cmd
}

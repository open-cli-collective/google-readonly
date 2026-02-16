// Package contacts implements the gro contacts command and subcommands.
package contacts

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the contacts parent command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"ppl"},
		Short:   "Google Contacts commands",
		Long: `Commands for reading Google Contacts.

All commands are read-only - no modifications are possible.

The short alias 'ppl' can be used instead of 'contacts':
  gro ppl list
  gro ppl search "John"
  gro ppl get <resource-name>
  gro ppl groups`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newSearchCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newGroupsCommand())

	return cmd
}

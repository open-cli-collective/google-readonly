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
		Long: `Commands for reading and organizing Google Contacts.

Supports read operations plus non-destructive organizational operations:
group membership management and starring.

The short alias 'ppl' can be used instead of 'contacts':
  gro ppl list
  gro ppl search "John"
  gro ppl get <resource-name>
  gro ppl groups
  gro ppl star <contact-id>
  gro ppl add-to-group "Friends" <contact-id>`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newSearchCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newGroupsCommand())
	cmd.AddCommand(newAddToGroupCommand())
	cmd.AddCommand(newRemoveFromGroupCommand())
	cmd.AddCommand(newStarCommand())
	cmd.AddCommand(newUnstarCommand())

	return cmd
}

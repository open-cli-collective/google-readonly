// Package mail implements the gro mail command and subcommands.
package mail

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the mail parent command with subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Gmail commands",
		Long: `Read-only access to Gmail messages, threads, and attachments.

This command group provides Gmail functionality:
- search: Search for messages using Gmail query syntax
- read: Read a single message
- thread: Read a full conversation thread
- labels: List all labels
- attachments: List and download attachments

Examples:
  gro mail search "is:unread"
  gro mail read <message-id>
  gro mail thread <thread-id>
  gro mail labels
  gro mail attachments list <message-id>`,
	}

	cmd.AddCommand(newSearchCommand())
	cmd.AddCommand(newReadCommand())
	cmd.AddCommand(newThreadCommand())
	cmd.AddCommand(newLabelsCommand())
	cmd.AddCommand(newAttachmentsCommand())

	return cmd
}

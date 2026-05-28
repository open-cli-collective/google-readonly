package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/format"
)

func newListAttachmentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <message-id>",
		Short: "List attachments in a message",
		Long: `List all attachments in a Gmail message with their metadata.

Shows filename, MIME type, size, and whether the attachment is inline.

Examples:
  gro mail attachments list 18abc123def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			attachments, err := client.GetAttachments(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("getting attachments: %w", err)
			}

			if len(attachments) == 0 {
				fmt.Println("No attachments found for message.")
				return nil
			}

			fmt.Printf("Found %d attachment(s):\n\n", len(attachments))
			for i, att := range attachments {
				// Sanitize filename to prevent terminal injection from malicious attachment names
				fmt.Printf("%d. %s\n", i+1, SanitizeFilename(att.Filename))
				fmt.Printf("   Type: %s\n", att.MimeType)
				fmt.Printf("   Size: %s\n", format.Size(att.Size))
				if att.IsInline {
					fmt.Printf("   Inline: yes\n")
				}
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

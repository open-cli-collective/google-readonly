package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListAttachmentsCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list <message-id>",
		Short: "List attachments in a message",
		Long: `List all attachments in a Gmail message with their metadata.

Shows filename, MIME type, size, and whether the attachment is inline.

Examples:
  gro mail attachments list 18abc123def456
  gro mail attachments list 18abc123def456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient()
			if err != nil {
				return fmt.Errorf("failed to create Gmail client: %w", err)
			}

			attachments, err := client.GetAttachments(args[0])
			if err != nil {
				return fmt.Errorf("failed to get attachments: %w", err)
			}

			if len(attachments) == 0 {
				fmt.Println("No attachments found for message.")
				return nil
			}

			if jsonOutput {
				return printJSON(attachments)
			}

			fmt.Printf("Found %d attachment(s):\n\n", len(attachments))
			for i, att := range attachments {
				// Sanitize filename to prevent terminal injection from malicious attachment names
				fmt.Printf("%d. %s\n", i+1, SanitizeFilename(att.Filename))
				fmt.Printf("   Type: %s\n", att.MimeType)
				fmt.Printf("   Size: %s\n", formatSize(att.Size))
				if att.IsInline {
					fmt.Printf("   Inline: yes\n")
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

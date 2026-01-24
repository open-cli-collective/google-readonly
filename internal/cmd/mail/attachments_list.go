package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listAttachmentsJSON bool

func newListAttachmentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <message-id>",
		Short: "List attachments in a message",
		Long: `List all attachments in a Gmail message with their metadata.

Shows filename, MIME type, size, and whether the attachment is inline.

Examples:
  gro mail attachments list 18abc123def456
  gro mail attachments list 18abc123def456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: runListAttachments,
	}

	cmd.Flags().BoolVarP(&listAttachmentsJSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runListAttachments(cmd *cobra.Command, args []string) error {
	client, err := newGmailClient()
	if err != nil {
		return err
	}

	attachments, err := client.GetAttachments(args[0])
	if err != nil {
		return err
	}

	if len(attachments) == 0 {
		fmt.Println("No attachments found for message.")
		return nil
	}

	if listAttachmentsJSON {
		return printJSON(attachments)
	}

	fmt.Printf("Found %d attachment(s):\n\n", len(attachments))
	for i, att := range attachments {
		fmt.Printf("%d. %s\n", i+1, att.Filename)
		fmt.Printf("   Type: %s\n", att.MimeType)
		fmt.Printf("   Size: %s\n", formatSize(att.Size))
		if att.IsInline {
			fmt.Printf("   Inline: yes\n")
		}
		fmt.Println()
	}

	return nil
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

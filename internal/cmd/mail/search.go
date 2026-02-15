package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	var (
		maxResults int64
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for messages",
		Long: `Search for Gmail messages using Gmail's search syntax.

Examples:
  gro mail search "from:alice@example.com"
  gro mail search "subject:meeting" --max 20
  gro mail search "is:unread" --json
  gro mail search "after:2024/01/01 before:2024/02/01"

For more query operators, see: https://support.google.com/mail/answer/7190`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			messages, skipped, err := client.SearchMessages(args[0], maxResults)
			if err != nil {
				return fmt.Errorf("searching messages: %w", err)
			}

			if len(messages) == 0 {
				fmt.Println("No messages found.")
				return nil
			}

			if jsonOutput {
				return printJSON(messages)
			}

			for _, msg := range messages {
				printMessageHeader(msg, MessagePrintOptions{
					IncludeThreadID: true,
					IncludeSnippet:  true,
				})
				fmt.Println("---")
			}

			if skipped > 0 {
				fmt.Printf("Note: %d message(s) could not be retrieved.\n", skipped)
			}

			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 10, "Maximum number of results to return")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

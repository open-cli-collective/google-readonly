package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReadCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "read <message-id>",
		Short: "Read a single message",
		Long: `Read the full content of a Gmail message by its ID.

The message ID can be obtained from the search command output.

Examples:
  gro mail read 18abc123def456
  gro mail read 18abc123def456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			msg, err := client.GetMessage(cmd.Context(), args[0], true)
			if err != nil {
				return fmt.Errorf("reading message: %w", err)
			}

			if jsonOutput {
				return printJSON(msg)
			}

			printMessageHeader(msg, MessagePrintOptions{
				IncludeTo:   true,
				IncludeBody: true,
			})

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output result as JSON")

	return cmd
}

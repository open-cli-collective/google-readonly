package mail

import (
	"github.com/spf13/cobra"
)

var readJSONOutput bool

func newReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <message-id>",
		Short: "Read a single message",
		Long: `Read the full content of a Gmail message by its ID.

The message ID can be obtained from the search command output.

Examples:
  gro mail read 18abc123def456
  gro mail read 18abc123def456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: runRead,
	}

	cmd.Flags().BoolVarP(&readJSONOutput, "json", "j", false, "Output result as JSON")

	return cmd
}

func runRead(cmd *cobra.Command, args []string) error {
	client, err := newGmailClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(args[0], true)
	if err != nil {
		return err
	}

	if readJSONOutput {
		return printJSON(msg)
	}

	printMessageHeader(msg, MessagePrintOptions{
		IncludeTo:   true,
		IncludeBody: true,
	})

	return nil
}

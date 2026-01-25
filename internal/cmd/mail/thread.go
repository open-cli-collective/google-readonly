package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

var threadJSONOutput bool

func newThreadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread <id>",
		Short: "Read a full conversation thread",
		Long: `Read all messages in a Gmail conversation thread.

Accepts either a thread ID or a message ID. If a message ID is provided,
the thread containing that message will be retrieved automatically.
Use the search command to find message IDs (the ThreadID field can also
be used directly).

Examples:
  gro mail thread 18abc123def456
  gro mail thread 18abc123def456 --json`,
		Args: cobra.ExactArgs(1),
		RunE: runThread,
	}

	cmd.Flags().BoolVarP(&threadJSONOutput, "json", "j", false, "Output result as JSON")

	return cmd
}

func runThread(cmd *cobra.Command, args []string) error {
	client, err := newGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	messages, err := client.GetThread(args[0])
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found in thread.")
		return nil
	}

	if threadJSONOutput {
		return printJSON(messages)
	}

	fmt.Printf("Thread contains %d message(s)\n\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("=== Message %d of %d ===\n", i+1, len(messages))
		printMessageHeader(msg, MessagePrintOptions{
			IncludeTo:   true,
			IncludeBody: true,
		})
		fmt.Println()
	}

	return nil
}

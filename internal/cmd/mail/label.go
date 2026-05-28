package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

func newLabelCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "label <label-name> [message-ids...]",
		Short: "Add a label to messages",
		Long: `Add a user label to Gmail messages. The label is resolved by display name.

Examples:
  gro mail label "Work" msg123 msg456
  gro mail search "from:boss" --ids | gro mail label "Important" --stdin
  gro mail label "Projects" --query "subject:sprint"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelName := args[0]
			messageArgs := args[1:]

			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ctx := cmd.Context()
			labelID, err := client.GetLabelID(ctx, labelName)
			if err != nil {
				return fmt.Errorf("resolving label: %w", err)
			}

			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  messageArgs,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchMessageIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:  fmt.Sprintf("added label '%s' to", labelName),
				IDs:     ids,
				Count:   len(ids),
				DryRun:  dryRun,
				Details: map[string]string{"label": labelName, "labelId": labelID},
			}

			if dryRun {
				result.Action = fmt.Sprintf("add label '%s' to", labelName)
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, []string{labelID}, nil); err != nil {
				return fmt.Errorf("adding label: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

func newUnlabelCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "unlabel <label-name> [message-ids...]",
		Short: "Remove a label from messages",
		Long: `Remove a user label from Gmail messages. The label is resolved by display name.

Examples:
  gro mail unlabel "Work" msg123
  gro mail unlabel "Old" --query "label:Old older_than:90d"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelName := args[0]
			messageArgs := args[1:]

			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ctx := cmd.Context()
			labelID, err := client.GetLabelID(ctx, labelName)
			if err != nil {
				return fmt.Errorf("resolving label: %w", err)
			}

			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  messageArgs,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchMessageIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:  fmt.Sprintf("removed label '%s' from", labelName),
				IDs:     ids,
				Count:   len(ids),
				DryRun:  dryRun,
				Details: map[string]string{"label": labelName, "labelId": labelID},
			}

			if dryRun {
				result.Action = fmt.Sprintf("remove label '%s' from", labelName)
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, nil, []string{labelID}); err != nil {
				return fmt.Errorf("removing label: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

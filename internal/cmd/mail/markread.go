package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

func newMarkReadCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "mark-read [message-ids...]",
		Short: "Mark messages as read",
		Long: `Mark Gmail messages as read by removing the UNREAD label.

Examples:
  gro mail mark-read msg123 msg456
  gro mail search "is:unread" --ids | gro mail mark-read --stdin
  gro mail mark-read --query "is:unread older_than:7d"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchMessageIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action: "marked as read",
				IDs:    ids,
				Count:  len(ids),
				DryRun: dryRun,
			}

			if dryRun {
				result.Action = "mark as read"
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, nil, []string{"UNREAD"}); err != nil {
				return fmt.Errorf("marking messages as read: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

func newMarkUnreadCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "mark-unread [message-ids...]",
		Short: "Mark messages as unread",
		Long: `Mark Gmail messages as unread by adding the UNREAD label.

Examples:
  gro mail mark-unread msg123
  gro mail mark-unread --query "subject:important"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchMessageIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action: "marked as unread",
				IDs:    ids,
				Count:  len(ids),
				DryRun: dryRun,
			}

			if dryRun {
				result.Action = "mark as unread"
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, []string{"UNREAD"}, nil); err != nil {
				return fmt.Errorf("marking messages as unread: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

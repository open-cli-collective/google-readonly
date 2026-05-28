package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

func newStarCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "star [message-ids...]",
		Short: "Star messages",
		Long: `Star Gmail messages by adding the STARRED label.

Examples:
  gro mail star msg123
  gro mail search "is:inbox" --ids | gro mail star --stdin
  gro mail star --query "from:boss is:unread"`,
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
				Action: "starred",
				IDs:    ids,
				Count:  len(ids),
				DryRun: dryRun,
			}

			if dryRun {
				result.Action = "star"
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, []string{"STARRED"}, nil); err != nil {
				return fmt.Errorf("starring messages: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

func newUnstarCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "unstar [message-ids...]",
		Short: "Unstar messages",
		Long: `Unstar Gmail messages by removing the STARRED label.

Examples:
  gro mail unstar msg123
  gro mail search "is:starred" --ids | gro mail unstar --stdin`,
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
				Action: "unstarred",
				IDs:    ids,
				Count:  len(ids),
				DryRun: dryRun,
			}

			if dryRun {
				result.Action = "unstar"
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, nil, []string{"STARRED"}); err != nil {
				return fmt.Errorf("unstarring messages: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

func newArchiveCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "archive [message-ids...]",
		Short: "Archive messages (remove from inbox)",
		Long: `Archive Gmail messages by removing the INBOX label.

Messages can be specified as positional arguments, piped via --stdin, or
resolved from a search query via --query. Only one input mode is allowed.

Examples:
  gro mail archive msg123 msg456
  gro mail search "from:newsletter" --ids | gro mail archive --stdin
  gro mail archive --query "from:noreply older_than:30d"
  gro mail archive --query "is:inbox from:noreply" --dry-run`,
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
				Action: "archived",
				IDs:    ids,
				Count:  len(ids),
				DryRun: dryRun,
			}

			if dryRun {
				result.Action = "archive"
				return result.Print()
			}

			if err := client.ModifyMessages(ctx, ids, nil, []string{"INBOX"}); err != nil {
				return fmt.Errorf("archiving messages: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

package drive

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
		Use:   "star [file-ids...]",
		Short: "Star files",
		Long: `Star files in Google Drive.

Supports three input modes (mutually exclusive):
  1. Positional arguments: gro drive star file1 file2
  2. Stdin (--stdin):      gro drive search "report" --ids | gro drive star --stdin
  3. Query (--query):      gro drive star --query "name contains 'report'"

Examples:
  gro drive star abc123
  gro drive search "budget" --ids | gro drive star --stdin
  gro drive star --query "name contains 'report'" --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchFileIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   "starred",
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "file",
			}

			if dryRun {
				result.Action = "star"
				return result.Print()
			}

			for _, id := range ids {
				if err := client.StarFile(ctx, id); err != nil {
					return fmt.Errorf("starring file %s: %w", id, err)
				}
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read file IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve file IDs")

	return cmd
}

func newUnstarCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "unstar [file-ids...]",
		Short: "Unstar files",
		Long: `Remove stars from files in Google Drive.

Supports three input modes (mutually exclusive):
  1. Positional arguments: gro drive unstar file1 file2
  2. Stdin (--stdin):      echo "file1" | gro drive unstar --stdin
  3. Query (--query):      gro drive unstar --query "starred = true"

Examples:
  gro drive unstar abc123
  gro drive unstar --query "starred = true" --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchFileIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   "unstarred",
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "file",
			}

			if dryRun {
				result.Action = "unstar"
				return result.Print()
			}

			for _, id := range ids {
				if err := client.UnstarFile(ctx, id); err != nil {
					return fmt.Errorf("unstarring file %s: %w", id, err)
				}
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read file IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve file IDs")

	return cmd
}

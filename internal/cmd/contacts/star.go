package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

// starredGroupResourceName is the system contact group for starred contacts.
const starredGroupResourceName = "contactGroups/starred"

func newStarCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "star [contact-ids...]",
		Short: "Star contacts",
		Long: `Star contacts by adding them to the system "Starred" group.

Contact IDs are resource names in the form "people/c1234567890".

Supports three input modes (mutually exclusive):
  1. Positional arguments: gro contacts star people/c123 people/c456
  2. Stdin (--stdin):      gro ppl search "John" --ids | gro contacts star --stdin
  3. Query (--query):      gro contacts star --query "John"

Examples:
  gro contacts star people/c1234567890
  gro ppl list --max 5 --ids | gro contacts star --stdin
  gro contacts star --query "John" --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchContactIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   "starred",
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "contact",
			}

			if dryRun {
				result.Action = "star"
				return result.Print()
			}

			if err := client.AddToGroup(ctx, starredGroupResourceName, ids); err != nil {
				return fmt.Errorf("starring contacts: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read contact IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve contact IDs")

	return cmd
}

func newUnstarCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "unstar [contact-ids...]",
		Short: "Unstar contacts",
		Long: `Remove stars from contacts by removing them from the system "Starred" group.

Contact IDs are resource names in the form "people/c1234567890".

Supports three input modes (mutually exclusive):
  1. Positional arguments: gro contacts unstar people/c123 people/c456
  2. Stdin (--stdin):      echo "people/c123" | gro contacts unstar --stdin
  3. Query (--query):      gro contacts unstar --query "John"

Examples:
  gro contacts unstar people/c1234567890
  gro contacts unstar --query "John" --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  args,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchContactIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   "unstarred",
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "contact",
			}

			if dryRun {
				result.Action = "unstar"
				return result.Print()
			}

			if err := client.RemoveFromGroup(ctx, starredGroupResourceName, ids); err != nil {
				return fmt.Errorf("unstarring contacts: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read contact IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve contact IDs")

	return cmd
}

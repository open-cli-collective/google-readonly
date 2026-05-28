package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

func newAddToGroupCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "add-to-group <group-name> [contact-ids...]",
		Short: "Add contacts to a group",
		Long: `Add contacts to a contact group.

The first argument is the group name (e.g., "Friends", "Work").
Contact IDs are resource names in the form "people/c1234567890".

Supports three input modes for contact IDs (mutually exclusive):
  1. Positional arguments: gro contacts add-to-group "Friends" people/c123 people/c456
  2. Stdin (--stdin):      gro ppl search "John" --ids | gro contacts add-to-group "Friends" --stdin
  3. Query (--query):      gro contacts add-to-group "Friends" --query "John"

Examples:
  gro contacts add-to-group "Friends" people/c1234567890
  gro ppl search "John" --ids | gro contacts add-to-group "Work" --stdin
  gro contacts add-to-group "Friends" --query "John" --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := args[0]
			contactArgs := args[1:]

			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			ctx := cmd.Context()

			groupResourceName, err := client.ResolveGroupName(ctx, groupName)
			if err != nil {
				return fmt.Errorf("resolving group: %w", err)
			}

			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  contactArgs,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchContactIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   fmt.Sprintf("added to group '%s'", groupName),
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "contact",
			}

			if dryRun {
				result.Action = fmt.Sprintf("add to group '%s'", groupName)
				return result.Print()
			}

			if err := client.AddToGroup(ctx, groupResourceName, ids); err != nil {
				return fmt.Errorf("adding contacts to group: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read contact IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve contact IDs")

	return cmd
}

func newRemoveFromGroupCommand() *cobra.Command {
	var (
		dryRun bool
		stdin  bool
		query  string
	)

	cmd := &cobra.Command{
		Use:   "remove-from-group <group-name> [contact-ids...]",
		Short: "Remove contacts from a group",
		Long: `Remove contacts from a contact group.

The first argument is the group name (e.g., "Friends", "Work").
Contact IDs are resource names in the form "people/c1234567890".

Supports three input modes for contact IDs (mutually exclusive):
  1. Positional arguments: gro contacts remove-from-group "Friends" people/c123 people/c456
  2. Stdin (--stdin):      echo "people/c123" | gro contacts remove-from-group "Friends" --stdin
  3. Query (--query):      gro contacts remove-from-group "Friends" --query "John"

Examples:
  gro contacts remove-from-group "Friends" people/c1234567890
  gro contacts remove-from-group "Work" --query "John" --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := args[0]
			contactArgs := args[1:]

			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			ctx := cmd.Context()

			groupResourceName, err := client.ResolveGroupName(ctx, groupName)
			if err != nil {
				return fmt.Errorf("resolving group: %w", err)
			}

			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  contactArgs,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchContactIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:   fmt.Sprintf("removed from group '%s'", groupName),
				IDs:      ids,
				Count:    len(ids),
				DryRun:   dryRun,
				ItemNoun: "contact",
			}

			if dryRun {
				result.Action = fmt.Sprintf("remove from group '%s'", groupName)
				return result.Print()
			}

			if err := client.RemoveFromGroup(ctx, groupResourceName, ids); err != nil {
				return fmt.Errorf("removing contacts from group: %w", err)
			}

			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read contact IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve contact IDs")

	return cmd
}

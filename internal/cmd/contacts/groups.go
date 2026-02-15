package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

func newGroupsCommand() *cobra.Command {
	var (
		maxResults int64
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List contact groups",
		Long: `List all contact groups (labels) from your Google Contacts.

Contact groups include both user-created labels and system groups.

Examples:
  gro contacts groups
  gro contacts groups --max 50
  gro ppl groups --json`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := newContactsClient()
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			resp, err := client.ListContactGroups("", maxResults)
			if err != nil {
				return fmt.Errorf("listing contact groups: %w", err)
			}

			if len(resp.ContactGroups) == 0 {
				fmt.Println("No contact groups found.")
				return nil
			}

			// Convert to our format
			parsedGroups := make([]*contacts.ContactGroup, len(resp.ContactGroups))
			for i, g := range resp.ContactGroups {
				parsedGroups[i] = contacts.ParseContactGroup(g)
			}

			if jsonOutput {
				return printJSON(parsedGroups)
			}

			fmt.Printf("Found %d contact group(s):\n\n", len(resp.ContactGroups))
			for _, group := range parsedGroups {
				printContactGroup(group)
			}

			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 30, "Maximum number of groups to return")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

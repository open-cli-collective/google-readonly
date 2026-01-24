package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

var (
	groupsMaxResults int64
	groupsJSONOutput bool
)

func newGroupsCommand() *cobra.Command {
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
		RunE: runGroups,
	}

	cmd.Flags().Int64VarP(&groupsMaxResults, "max", "m", 30, "Maximum number of groups to return")
	cmd.Flags().BoolVarP(&groupsJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runGroups(cmd *cobra.Command, args []string) error {
	client, err := newContactsClient()
	if err != nil {
		return err
	}

	resp, err := client.ListContactGroups("", groupsMaxResults)
	if err != nil {
		return err
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

	if groupsJSONOutput {
		return printJSON(parsedGroups)
	}

	fmt.Printf("Found %d contact group(s):\n\n", len(resp.ContactGroups))
	for _, group := range parsedGroups {
		printContactGroup(group)
	}

	return nil
}

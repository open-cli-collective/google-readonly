package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

var (
	listMaxResults int64
	listJSONOutput bool
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		Long: `List all contacts from your Google Contacts.

Contacts are sorted by last name.

Examples:
  gro contacts list
  gro contacts list --max 50
  gro ppl list --json`,
		Args: cobra.NoArgs,
		RunE: runList,
	}

	cmd.Flags().Int64VarP(&listMaxResults, "max", "m", 10, "Maximum number of contacts to return")
	cmd.Flags().BoolVarP(&listJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := newContactsClient()
	if err != nil {
		return err
	}

	resp, err := client.ListContacts("", listMaxResults)
	if err != nil {
		return err
	}

	if len(resp.Connections) == 0 {
		fmt.Println("No contacts found.")
		return nil
	}

	// Convert to our format
	parsedContacts := make([]*contacts.Contact, len(resp.Connections))
	for i, p := range resp.Connections {
		parsedContacts[i] = contacts.ParseContact(p)
	}

	if listJSONOutput {
		return printJSON(parsedContacts)
	}

	fmt.Printf("Found %d contact(s):\n\n", len(resp.Connections))
	for _, contact := range parsedContacts {
		printContactSummary(contact)
	}

	return nil
}

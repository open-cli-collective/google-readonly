package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

func newSearchCommand() *cobra.Command {
	var (
		maxResults int64
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search contacts",
		Long: `Search contacts by name, email, phone number, or organization.

The query is matched against multiple fields:
- Display name
- Given name and family name
- Email addresses
- Phone numbers
- Organization name

Examples:
  gro contacts search "John"
  gro contacts search "example.com"
  gro contacts search "+1-555" --max 20
  gro ppl search "Acme" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := args[0]

			client, err := newContactsClient()
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			resp, err := client.SearchContacts(query, maxResults)
			if err != nil {
				return fmt.Errorf("searching contacts: %w", err)
			}

			if len(resp.Results) == 0 {
				fmt.Printf("No contacts found matching \"%s\".\n", query)
				return nil
			}

			// Convert to our format
			parsedContacts := make([]*contacts.Contact, len(resp.Results))
			for i, r := range resp.Results {
				parsedContacts[i] = contacts.ParseContact(r.Person)
			}

			if jsonOutput {
				return printJSON(parsedContacts)
			}

			fmt.Printf("Found %d contact(s) matching \"%s\":\n\n", len(resp.Results), query)
			for _, contact := range parsedContacts {
				printContactSummary(contact)
			}

			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 10, "Maximum number of results")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

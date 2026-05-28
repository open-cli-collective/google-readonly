package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

func newListCommand() *cobra.Command {
	var (
		maxResults int64
		idsOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		Long: `List all contacts from your Google Contacts.

Contacts are sorted by last name.

Examples:
  gro contacts list
  gro contacts list --max 50
  gro ppl list --ids | gro contacts star --stdin`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			resp, err := client.ListContacts(cmd.Context(), "", maxResults)
			if err != nil {
				return fmt.Errorf("listing contacts: %w", err)
			}

			if len(resp.Connections) == 0 {
				if !idsOutput {
					fmt.Println("No contacts found.")
				}
				return nil
			}

			if idsOutput {
				for _, p := range resp.Connections {
					fmt.Println(p.ResourceName)
				}
				return nil
			}

			parsedContacts := make([]*contacts.Contact, len(resp.Connections))
			for i, p := range resp.Connections {
				parsedContacts[i] = contacts.ParseContact(p)
			}

			fmt.Printf("Found %d contact(s):\n\n", len(resp.Connections))
			for _, contact := range parsedContacts {
				printContactSummary(contact)
			}

			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 10, "Maximum number of contacts to return")
	cmd.Flags().BoolVar(&idsOutput, "ids", false, "Output only resource names, one per line")

	return cmd
}

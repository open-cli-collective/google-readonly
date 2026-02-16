package contacts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
)

func newGetCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get <resource-name>",
		Short: "Get contact details",
		Long: `Get the full details of a specific contact.

The resource name is in the format "people/c123456789" and can be
obtained from the list or search commands.

Examples:
  gro contacts get people/c123456789
  gro ppl get people/c123456789 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resourceName := args[0]

			client, err := newContactsClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}

			person, err := client.GetContact(cmd.Context(), resourceName)
			if err != nil {
				return fmt.Errorf("getting contact: %w", err)
			}

			contact := contacts.ParseContact(person)

			if jsonOutput {
				return printJSON(contact)
			}

			printContact(contact, true)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

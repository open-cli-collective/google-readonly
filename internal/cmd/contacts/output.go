package contacts

import (
	"context"
	"fmt"

	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// ContactsClient defines the interface for Contacts client operations used by contacts commands.
type ContactsClient interface {
	ListContacts(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error)
	SearchContacts(query string, pageSize int64) (*people.SearchResponse, error)
	GetContact(resourceName string) (*people.Person, error)
	ListContactGroups(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error)
}

// ClientFactory is the function used to create Contacts clients.
// Override in tests to inject mocks.
var ClientFactory = func() (ContactsClient, error) {
	return contacts.NewClient(context.Background())
}

// newContactsClient creates a new contacts client
func newContactsClient() (ContactsClient, error) {
	return ClientFactory()
}

// printJSON outputs data as indented JSON
func printJSON(data any) error {
	return output.JSONStdout(data)
}

// printContact prints a single contact in text format
func printContact(contact *contacts.Contact, showDetails bool) {
	fmt.Printf("ID: %s\n", contact.ResourceName)
	fmt.Printf("Name: %s\n", contact.GetDisplayName())

	if email := contact.GetPrimaryEmail(); email != "" {
		fmt.Printf("Email: %s\n", email)
	}

	if phone := contact.GetPrimaryPhone(); phone != "" {
		fmt.Printf("Phone: %s\n", phone)
	}

	if org := contact.GetOrganization(); org != "" {
		fmt.Printf("Organization: %s\n", org)
	}

	if showDetails {
		// Show all emails
		if len(contact.Emails) > 1 {
			fmt.Println("All Emails:")
			for _, e := range contact.Emails {
				primary := ""
				if e.Primary {
					primary = " (primary)"
				}
				typeStr := ""
				if e.Type != "" {
					typeStr = fmt.Sprintf(" [%s]", e.Type)
				}
				fmt.Printf("  - %s%s%s\n", e.Value, typeStr, primary)
			}
		}

		// Show all phones
		if len(contact.Phones) > 1 {
			fmt.Println("All Phones:")
			for _, p := range contact.Phones {
				typeStr := ""
				if p.Type != "" {
					typeStr = fmt.Sprintf(" [%s]", p.Type)
				}
				fmt.Printf("  - %s%s\n", p.Value, typeStr)
			}
		}

		// Show all organizations
		if len(contact.Organizations) > 0 {
			fmt.Println("Organizations:")
			for _, o := range contact.Organizations {
				if o.Name != "" {
					fmt.Printf("  - %s", o.Name)
					if o.Title != "" {
						fmt.Printf(" (%s)", o.Title)
					}
					if o.Department != "" {
						fmt.Printf(" - %s", o.Department)
					}
					fmt.Println()
				} else if o.Title != "" {
					fmt.Printf("  - %s\n", o.Title)
				}
			}
		}

		// Show addresses
		if len(contact.Addresses) > 0 {
			fmt.Println("Addresses:")
			for _, a := range contact.Addresses {
				typeStr := ""
				if a.Type != "" {
					typeStr = fmt.Sprintf("[%s] ", a.Type)
				}
				fmt.Printf("  - %s%s\n", typeStr, a.FormattedValue)
			}
		}

		// Show URLs
		if len(contact.URLs) > 0 {
			fmt.Println("URLs:")
			for _, u := range contact.URLs {
				typeStr := ""
				if u.Type != "" {
					typeStr = fmt.Sprintf("[%s] ", u.Type)
				}
				fmt.Printf("  - %s%s\n", typeStr, u.Value)
			}
		}

		// Show birthday
		if contact.Birthday != "" {
			fmt.Printf("Birthday: %s\n", contact.Birthday)
		}

		// Show biography
		if contact.Biography != "" {
			fmt.Println()
			fmt.Println("--- Biography ---")
			fmt.Println(contact.Biography)
		}
	}
}

// printContactSummary prints a brief contact summary for list views
func printContactSummary(contact *contacts.Contact) {
	fmt.Printf("ID: %s\n", contact.ResourceName)
	fmt.Printf("Name: %s\n", contact.GetDisplayName())

	if email := contact.GetPrimaryEmail(); email != "" {
		fmt.Printf("Email: %s\n", email)
	}

	if phone := contact.GetPrimaryPhone(); phone != "" {
		fmt.Printf("Phone: %s\n", phone)
	}

	if org := contact.GetOrganization(); org != "" {
		fmt.Printf("Organization: %s\n", org)
	}

	fmt.Println("---")
}

// printContactGroup prints a contact group
func printContactGroup(group *contacts.ContactGroup) {
	fmt.Printf("ID: %s\n", group.ResourceName)
	fmt.Printf("Name: %s\n", group.Name)
	if group.GroupType != "" {
		fmt.Printf("Type: %s\n", group.GroupType)
	}
	fmt.Printf("Members: %d\n", group.MemberCount)
	fmt.Println("---")
}

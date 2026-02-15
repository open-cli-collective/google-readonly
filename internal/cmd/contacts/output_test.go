package contacts

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/contacts"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{
			name: "single contact",
			data: &contacts.Contact{
				ResourceName: "people/c123",
				DisplayName:  "John Doe",
			},
		},
		{
			name: "contact list",
			data: []*contacts.Contact{
				{ResourceName: "people/c1", DisplayName: "Alice"},
				{ResourceName: "people/c2", DisplayName: "Bob"},
			},
		},
		{
			name: "contact group",
			data: &contacts.ContactGroup{
				ResourceName: "contactGroups/abc",
				Name:         "Work",
				MemberCount:  5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := printJSON(tt.data)
			testutil.NoError(t, err)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()
			testutil.NotEmpty(t, output)

			// Verify it's valid JSON
			var parsed any
			err = json.Unmarshal([]byte(output), &parsed)
			testutil.NoError(t, err)
		})
	}
}

func TestPrintContact(t *testing.T) {
	tests := []struct {
		name            string
		contact         *contacts.Contact
		showDetails     bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "basic contact",
			contact: &contacts.Contact{
				ResourceName: "people/c123",
				DisplayName:  "John Doe",
			},
			showDetails: false,
			wantContains: []string{
				"ID: people/c123",
				"Name: John Doe",
			},
		},
		{
			name: "contact with email",
			contact: &contacts.Contact{
				ResourceName: "people/c456",
				DisplayName:  "Jane Smith",
				Emails: []contacts.Email{
					{Value: "jane@example.com", Primary: true},
				},
			},
			showDetails: false,
			wantContains: []string{
				"Email: jane@example.com",
			},
		},
		{
			name: "contact with phone",
			contact: &contacts.Contact{
				ResourceName: "people/c789",
				DisplayName:  "Bob Wilson",
				Phones: []contacts.Phone{
					{Value: "+1-555-123-4567", Type: "mobile"},
				},
			},
			showDetails: false,
			wantContains: []string{
				"Phone: +1-555-123-4567",
			},
		},
		{
			name: "contact with organization",
			contact: &contacts.Contact{
				ResourceName: "people/c101",
				DisplayName:  "Alice Brown",
				Organizations: []contacts.Organization{
					{Name: "Acme Corp", Title: "Engineer"},
				},
			},
			showDetails: false,
			wantContains: []string{
				"Organization: Acme Corp",
			},
		},
		{
			name: "contact with details - multiple emails",
			contact: &contacts.Contact{
				ResourceName: "people/c102",
				DisplayName:  "Charlie Davis",
				Emails: []contacts.Email{
					{Value: "charlie@work.com", Type: "work", Primary: true},
					{Value: "charlie@home.com", Type: "home"},
				},
			},
			showDetails: true,
			wantContains: []string{
				"All Emails:",
				"charlie@work.com",
				"[work]",
				"(primary)",
				"charlie@home.com",
			},
		},
		{
			name: "contact with details - addresses",
			contact: &contacts.Contact{
				ResourceName: "people/c103",
				DisplayName:  "Diana Evans",
				Addresses: []contacts.Address{
					{FormattedValue: "123 Main St, SF, CA", Type: "home"},
				},
			},
			showDetails: true,
			wantContains: []string{
				"Addresses:",
				"[home]",
				"123 Main St",
			},
		},
		{
			name: "contact with details - URLs",
			contact: &contacts.Contact{
				ResourceName: "people/c104",
				DisplayName:  "Eve Franklin",
				URLs: []contacts.URL{
					{Value: "https://linkedin.com/in/eve", Type: "profile"},
				},
			},
			showDetails: true,
			wantContains: []string{
				"URLs:",
				"https://linkedin.com/in/eve",
			},
		},
		{
			name: "contact with details - birthday",
			contact: &contacts.Contact{
				ResourceName: "people/c105",
				DisplayName:  "Frank Garcia",
				Birthday:     "1990-06-15",
			},
			showDetails: true,
			wantContains: []string{
				"Birthday: 1990-06-15",
			},
		},
		{
			name: "contact with details - biography",
			contact: &contacts.Contact{
				ResourceName: "people/c106",
				DisplayName:  "Grace Harris",
				Biography:    "Software engineer and open source contributor.",
			},
			showDetails: true,
			wantContains: []string{
				"--- Biography ---",
				"Software engineer",
			},
		},
		{
			name: "contact without details - hides biography",
			contact: &contacts.Contact{
				ResourceName: "people/c107",
				DisplayName:  "Henry Irving",
				Biography:    "Should be hidden",
			},
			showDetails: false,
			wantNotContains: []string{
				"--- Biography ---",
				"Should be hidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printContact(tt.contact, tt.showDetails)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}
			for _, notWant := range tt.wantNotContains {
				testutil.NotContains(t, output, notWant)
			}
		})
	}
}

func TestPrintContactSummary(t *testing.T) {
	tests := []struct {
		name         string
		contact      *contacts.Contact
		wantContains []string
	}{
		{
			name: "basic summary",
			contact: &contacts.Contact{
				ResourceName: "people/c123",
				DisplayName:  "John Doe",
				Emails:       []contacts.Email{{Value: "john@example.com"}},
				Phones:       []contacts.Phone{{Value: "+1-555-1234"}},
			},
			wantContains: []string{
				"ID: people/c123",
				"Name: John Doe",
				"Email: john@example.com",
				"Phone: +1-555-1234",
				"---",
			},
		},
		{
			name: "summary with organization",
			contact: &contacts.Contact{
				ResourceName: "people/c456",
				DisplayName:  "Jane Smith",
				Organizations: []contacts.Organization{
					{Name: "Tech Corp"},
				},
			},
			wantContains: []string{
				"Organization: Tech Corp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printContactSummary(tt.contact)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}
		})
	}
}

func TestPrintContactGroup(t *testing.T) {
	tests := []struct {
		name         string
		group        *contacts.ContactGroup
		wantContains []string
	}{
		{
			name: "user contact group",
			group: &contacts.ContactGroup{
				ResourceName: "contactGroups/abc123",
				Name:         "Work",
				GroupType:    "USER_CONTACT_GROUP",
				MemberCount:  42,
			},
			wantContains: []string{
				"ID: contactGroups/abc123",
				"Name: Work",
				"Type: USER_CONTACT_GROUP",
				"Members: 42",
				"---",
			},
		},
		{
			name: "system group",
			group: &contacts.ContactGroup{
				ResourceName: "contactGroups/all",
				Name:         "All Contacts",
				GroupType:    "SYSTEM_CONTACT_GROUP",
				MemberCount:  100,
			},
			wantContains: []string{
				"Name: All Contacts",
				"Type: SYSTEM_CONTACT_GROUP",
				"Members: 100",
			},
		},
		{
			name: "group without type",
			group: &contacts.ContactGroup{
				ResourceName: "contactGroups/xyz",
				Name:         "Friends",
				MemberCount:  5,
			},
			wantContains: []string{
				"Name: Friends",
				"Members: 5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printContactGroup(tt.group)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()

			for _, want := range tt.wantContains {
				testutil.Contains(t, output, want)
			}
		})
	}
}

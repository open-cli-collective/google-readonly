package contacts

import (
	"testing"

	"google.golang.org/api/people/v1"
)

func TestParseContact(t *testing.T) {
	t.Run("parses basic contact", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c123",
			Names: []*people.Name{
				{
					DisplayName: "John Doe",
					GivenName:   "John",
					FamilyName:  "Doe",
				},
			},
		}

		contact := ParseContact(p)

		if contact.ResourceName != "people/c123" {
			t.Errorf("got %v, want %v", contact.ResourceName, "people/c123")
		}
		if contact.DisplayName != "John Doe" {
			t.Errorf("got %v, want %v", contact.DisplayName, "John Doe")
		}
		if len(contact.Names) != 1 {
			t.Errorf("got length %d, want %d", len(contact.Names), 1)
		}
		if contact.Names[0].GivenName != "John" {
			t.Errorf("got %v, want %v", contact.Names[0].GivenName, "John")
		}
		if contact.Names[0].FamilyName != "Doe" {
			t.Errorf("got %v, want %v", contact.Names[0].FamilyName, "Doe")
		}
	})

	t.Run("parses contact with email", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c456",
			Names: []*people.Name{
				{DisplayName: "Jane Smith"},
			},
			EmailAddresses: []*people.EmailAddress{
				{
					Value:    "jane@example.com",
					Type:     "work",
					Metadata: &people.FieldMetadata{Primary: true},
				},
				{
					Value: "jane.personal@example.com",
					Type:  "home",
				},
			},
		}

		contact := ParseContact(p)

		if len(contact.Emails) != 2 {
			t.Errorf("got length %d, want %d", len(contact.Emails), 2)
		}
		if contact.Emails[0].Value != "jane@example.com" {
			t.Errorf("got %v, want %v", contact.Emails[0].Value, "jane@example.com")
		}
		if contact.Emails[0].Type != "work" {
			t.Errorf("got %v, want %v", contact.Emails[0].Type, "work")
		}
		if !contact.Emails[0].Primary {
			t.Error("got false, want true")
		}
		if contact.Emails[1].Value != "jane.personal@example.com" {
			t.Errorf("got %v, want %v", contact.Emails[1].Value, "jane.personal@example.com")
		}
		if contact.Emails[1].Primary {
			t.Error("got true, want false")
		}
	})

	t.Run("parses contact with phone numbers", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c789",
			PhoneNumbers: []*people.PhoneNumber{
				{Value: "+1-555-123-4567", Type: "mobile"},
				{Value: "+1-555-987-6543", Type: "work"},
			},
		}

		contact := ParseContact(p)

		if len(contact.Phones) != 2 {
			t.Errorf("got length %d, want %d", len(contact.Phones), 2)
		}
		if contact.Phones[0].Value != "+1-555-123-4567" {
			t.Errorf("got %v, want %v", contact.Phones[0].Value, "+1-555-123-4567")
		}
		if contact.Phones[0].Type != "mobile" {
			t.Errorf("got %v, want %v", contact.Phones[0].Type, "mobile")
		}
	})

	t.Run("parses contact with organization", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c101",
			Organizations: []*people.Organization{
				{
					Name:       "Acme Corp",
					Title:      "Software Engineer",
					Department: "Engineering",
				},
			},
		}

		contact := ParseContact(p)

		if len(contact.Organizations) != 1 {
			t.Errorf("got length %d, want %d", len(contact.Organizations), 1)
		}
		if contact.Organizations[0].Name != "Acme Corp" {
			t.Errorf("got %v, want %v", contact.Organizations[0].Name, "Acme Corp")
		}
		if contact.Organizations[0].Title != "Software Engineer" {
			t.Errorf("got %v, want %v", contact.Organizations[0].Title, "Software Engineer")
		}
		if contact.Organizations[0].Department != "Engineering" {
			t.Errorf("got %v, want %v", contact.Organizations[0].Department, "Engineering")
		}
	})

	t.Run("parses contact with address", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c102",
			Addresses: []*people.Address{
				{
					FormattedValue: "123 Main St, San Francisco, CA 94102",
					Type:           "home",
					City:           "San Francisco",
					Region:         "CA",
					PostalCode:     "94102",
					Country:        "USA",
				},
			},
		}

		contact := ParseContact(p)

		if len(contact.Addresses) != 1 {
			t.Errorf("got length %d, want %d", len(contact.Addresses), 1)
		}
		if contact.Addresses[0].Type != "home" {
			t.Errorf("got %v, want %v", contact.Addresses[0].Type, "home")
		}
		if contact.Addresses[0].City != "San Francisco" {
			t.Errorf("got %v, want %v", contact.Addresses[0].City, "San Francisco")
		}
		if contact.Addresses[0].PostalCode != "94102" {
			t.Errorf("got %v, want %v", contact.Addresses[0].PostalCode, "94102")
		}
	})

	t.Run("parses contact with URLs", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c103",
			Urls: []*people.Url{
				{Value: "https://linkedin.com/in/johndoe", Type: "profile"},
				{Value: "https://github.com/johndoe", Type: "other"},
			},
		}

		contact := ParseContact(p)

		if len(contact.URLs) != 2 {
			t.Errorf("got length %d, want %d", len(contact.URLs), 2)
		}
		if contact.URLs[0].Value != "https://linkedin.com/in/johndoe" {
			t.Errorf("got %v, want %v", contact.URLs[0].Value, "https://linkedin.com/in/johndoe")
		}
	})

	t.Run("parses contact with biography", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c104",
			Biographies: []*people.Biography{
				{Value: "A passionate software developer."},
			},
		}

		contact := ParseContact(p)

		if contact.Biography != "A passionate software developer." {
			t.Errorf("got %v, want %v", contact.Biography, "A passionate software developer.")
		}
	})

	t.Run("parses contact with birthday including year", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c105",
			Birthdays: []*people.Birthday{
				{Date: &people.Date{Year: 1990, Month: 6, Day: 15}},
			},
		}

		contact := ParseContact(p)

		if contact.Birthday != "1990-06-15" {
			t.Errorf("got %v, want %v", contact.Birthday, "1990-06-15")
		}
	})

	t.Run("parses contact with birthday month/day only", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c106",
			Birthdays: []*people.Birthday{
				{Date: &people.Date{Month: 12, Day: 25}},
			},
		}

		contact := ParseContact(p)

		if contact.Birthday != "12-25" {
			t.Errorf("got %v, want %v", contact.Birthday, "12-25")
		}
	})

	t.Run("parses contact with photo", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c107",
			Photos: []*people.Photo{
				{Url: "https://example.com/photo.jpg"},
			},
		}

		contact := ParseContact(p)

		if contact.PhotoURL != "https://example.com/photo.jpg" {
			t.Errorf("got %v, want %v", contact.PhotoURL, "https://example.com/photo.jpg")
		}
	})

	t.Run("handles nil person", func(t *testing.T) {
		contact := ParseContact(nil)
		if contact != nil {
			t.Errorf("got %v, want nil", contact)
		}
	})
}

func TestParseContactGroup(t *testing.T) {
	t.Run("parses contact group", func(t *testing.T) {
		g := &people.ContactGroup{
			ResourceName: "contactGroups/abc123",
			Name:         "Work",
			GroupType:    "USER_CONTACT_GROUP",
			MemberCount:  42,
		}

		group := ParseContactGroup(g)

		if group.ResourceName != "contactGroups/abc123" {
			t.Errorf("got %v, want %v", group.ResourceName, "contactGroups/abc123")
		}
		if group.Name != "Work" {
			t.Errorf("got %v, want %v", group.Name, "Work")
		}
		if group.GroupType != "USER_CONTACT_GROUP" {
			t.Errorf("got %v, want %v", group.GroupType, "USER_CONTACT_GROUP")
		}
		if group.MemberCount != int64(42) {
			t.Errorf("got %v, want %v", group.MemberCount, int64(42))
		}
	})

	t.Run("handles nil group", func(t *testing.T) {
		group := ParseContactGroup(nil)
		if group != nil {
			t.Errorf("got %v, want nil", group)
		}
	})
}

func TestContactGetDisplayName(t *testing.T) {
	t.Run("returns display name when set", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c1",
			DisplayName:  "John Doe",
		}
		if c.GetDisplayName() != "John Doe" {
			t.Errorf("got %v, want %v", c.GetDisplayName(), "John Doe")
		}
	})

	t.Run("falls back to names array", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c2",
			Names: []Name{
				{DisplayName: "Jane Smith"},
			},
		}
		if c.GetDisplayName() != "Jane Smith" {
			t.Errorf("got %v, want %v", c.GetDisplayName(), "Jane Smith")
		}
	})

	t.Run("falls back to email", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c3",
			Emails: []Email{
				{Value: "test@example.com"},
			},
		}
		if c.GetDisplayName() != "test@example.com" {
			t.Errorf("got %v, want %v", c.GetDisplayName(), "test@example.com")
		}
	})

	t.Run("falls back to resource name", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c4",
		}
		if c.GetDisplayName() != "people/c4" {
			t.Errorf("got %v, want %v", c.GetDisplayName(), "people/c4")
		}
	})
}

func TestContactGetPrimaryEmail(t *testing.T) {
	t.Run("returns primary email when marked", func(t *testing.T) {
		c := &Contact{
			Emails: []Email{
				{Value: "work@example.com", Primary: false},
				{Value: "primary@example.com", Primary: true},
			},
		}
		if c.GetPrimaryEmail() != "primary@example.com" {
			t.Errorf("got %v, want %v", c.GetPrimaryEmail(), "primary@example.com")
		}
	})

	t.Run("returns first email when no primary", func(t *testing.T) {
		c := &Contact{
			Emails: []Email{
				{Value: "first@example.com"},
				{Value: "second@example.com"},
			},
		}
		if c.GetPrimaryEmail() != "first@example.com" {
			t.Errorf("got %v, want %v", c.GetPrimaryEmail(), "first@example.com")
		}
	})

	t.Run("returns empty string when no emails", func(t *testing.T) {
		c := &Contact{}
		if c.GetPrimaryEmail() != "" {
			t.Errorf("got %v, want %v", c.GetPrimaryEmail(), "")
		}
	})
}

func TestContactGetPrimaryPhone(t *testing.T) {
	t.Run("returns first phone", func(t *testing.T) {
		c := &Contact{
			Phones: []Phone{
				{Value: "+1-555-123-4567"},
				{Value: "+1-555-987-6543"},
			},
		}
		if c.GetPrimaryPhone() != "+1-555-123-4567" {
			t.Errorf("got %v, want %v", c.GetPrimaryPhone(), "+1-555-123-4567")
		}
	})

	t.Run("returns empty string when no phones", func(t *testing.T) {
		c := &Contact{}
		if c.GetPrimaryPhone() != "" {
			t.Errorf("got %v, want %v", c.GetPrimaryPhone(), "")
		}
	})
}

func TestContactGetOrganization(t *testing.T) {
	t.Run("returns organization name", func(t *testing.T) {
		c := &Contact{
			Organizations: []Organization{
				{Name: "Acme Corp", Title: "Engineer"},
			},
		}
		if c.GetOrganization() != "Acme Corp" {
			t.Errorf("got %v, want %v", c.GetOrganization(), "Acme Corp")
		}
	})

	t.Run("returns title when no name", func(t *testing.T) {
		c := &Contact{
			Organizations: []Organization{
				{Title: "Freelance Developer"},
			},
		}
		if c.GetOrganization() != "Freelance Developer" {
			t.Errorf("got %v, want %v", c.GetOrganization(), "Freelance Developer")
		}
	})

	t.Run("returns empty string when no organizations", func(t *testing.T) {
		c := &Contact{}
		if c.GetOrganization() != "" {
			t.Errorf("got %v, want %v", c.GetOrganization(), "")
		}
	})
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name   string
		year   int64
		month  int64
		day    int64
		expect string
	}{
		{"full date", 2024, 12, 25, "2024-12-25"},
		{"single digit month/day", 2024, 1, 5, "2024-01-05"},
		{"missing components", 0, 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.year, tt.month, tt.day)
			if result != tt.expect {
				t.Errorf("got %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestFormatMonthDay(t *testing.T) {
	tests := []struct {
		name   string
		month  int64
		day    int64
		expect string
	}{
		{"standard date", 12, 25, "12-25"},
		{"single digit", 1, 5, "01-05"},
		{"missing components", 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMonthDay(tt.month, tt.day)
			if result != tt.expect {
				t.Errorf("got %v, want %v", result, tt.expect)
			}
		})
	}
}

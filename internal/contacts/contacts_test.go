package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

		assert.Equal(t, "people/c123", contact.ResourceName)
		assert.Equal(t, "John Doe", contact.DisplayName)
		assert.Len(t, contact.Names, 1)
		assert.Equal(t, "John", contact.Names[0].GivenName)
		assert.Equal(t, "Doe", contact.Names[0].FamilyName)
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

		assert.Len(t, contact.Emails, 2)
		assert.Equal(t, "jane@example.com", contact.Emails[0].Value)
		assert.Equal(t, "work", contact.Emails[0].Type)
		assert.True(t, contact.Emails[0].Primary)
		assert.Equal(t, "jane.personal@example.com", contact.Emails[1].Value)
		assert.False(t, contact.Emails[1].Primary)
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

		assert.Len(t, contact.Phones, 2)
		assert.Equal(t, "+1-555-123-4567", contact.Phones[0].Value)
		assert.Equal(t, "mobile", contact.Phones[0].Type)
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

		assert.Len(t, contact.Organizations, 1)
		assert.Equal(t, "Acme Corp", contact.Organizations[0].Name)
		assert.Equal(t, "Software Engineer", contact.Organizations[0].Title)
		assert.Equal(t, "Engineering", contact.Organizations[0].Department)
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

		assert.Len(t, contact.Addresses, 1)
		assert.Equal(t, "home", contact.Addresses[0].Type)
		assert.Equal(t, "San Francisco", contact.Addresses[0].City)
		assert.Equal(t, "94102", contact.Addresses[0].PostalCode)
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

		assert.Len(t, contact.URLs, 2)
		assert.Equal(t, "https://linkedin.com/in/johndoe", contact.URLs[0].Value)
	})

	t.Run("parses contact with biography", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c104",
			Biographies: []*people.Biography{
				{Value: "A passionate software developer."},
			},
		}

		contact := ParseContact(p)

		assert.Equal(t, "A passionate software developer.", contact.Biography)
	})

	t.Run("parses contact with birthday including year", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c105",
			Birthdays: []*people.Birthday{
				{Date: &people.Date{Year: 1990, Month: 6, Day: 15}},
			},
		}

		contact := ParseContact(p)

		assert.Equal(t, "1990-06-15", contact.Birthday)
	})

	t.Run("parses contact with birthday month/day only", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c106",
			Birthdays: []*people.Birthday{
				{Date: &people.Date{Month: 12, Day: 25}},
			},
		}

		contact := ParseContact(p)

		assert.Equal(t, "12-25", contact.Birthday)
	})

	t.Run("parses contact with photo", func(t *testing.T) {
		p := &people.Person{
			ResourceName: "people/c107",
			Photos: []*people.Photo{
				{Url: "https://example.com/photo.jpg"},
			},
		}

		contact := ParseContact(p)

		assert.Equal(t, "https://example.com/photo.jpg", contact.PhotoURL)
	})

	t.Run("handles nil person", func(t *testing.T) {
		contact := ParseContact(nil)
		assert.Nil(t, contact)
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

		assert.Equal(t, "contactGroups/abc123", group.ResourceName)
		assert.Equal(t, "Work", group.Name)
		assert.Equal(t, "USER_CONTACT_GROUP", group.GroupType)
		assert.Equal(t, int64(42), group.MemberCount)
	})

	t.Run("handles nil group", func(t *testing.T) {
		group := ParseContactGroup(nil)
		assert.Nil(t, group)
	})
}

func TestContactGetDisplayName(t *testing.T) {
	t.Run("returns display name when set", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c1",
			DisplayName:  "John Doe",
		}
		assert.Equal(t, "John Doe", c.GetDisplayName())
	})

	t.Run("falls back to names array", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c2",
			Names: []Name{
				{DisplayName: "Jane Smith"},
			},
		}
		assert.Equal(t, "Jane Smith", c.GetDisplayName())
	})

	t.Run("falls back to email", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c3",
			Emails: []Email{
				{Value: "test@example.com"},
			},
		}
		assert.Equal(t, "test@example.com", c.GetDisplayName())
	})

	t.Run("falls back to resource name", func(t *testing.T) {
		c := &Contact{
			ResourceName: "people/c4",
		}
		assert.Equal(t, "people/c4", c.GetDisplayName())
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
		assert.Equal(t, "primary@example.com", c.GetPrimaryEmail())
	})

	t.Run("returns first email when no primary", func(t *testing.T) {
		c := &Contact{
			Emails: []Email{
				{Value: "first@example.com"},
				{Value: "second@example.com"},
			},
		}
		assert.Equal(t, "first@example.com", c.GetPrimaryEmail())
	})

	t.Run("returns empty string when no emails", func(t *testing.T) {
		c := &Contact{}
		assert.Equal(t, "", c.GetPrimaryEmail())
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
		assert.Equal(t, "+1-555-123-4567", c.GetPrimaryPhone())
	})

	t.Run("returns empty string when no phones", func(t *testing.T) {
		c := &Contact{}
		assert.Equal(t, "", c.GetPrimaryPhone())
	})
}

func TestContactGetOrganization(t *testing.T) {
	t.Run("returns organization name", func(t *testing.T) {
		c := &Contact{
			Organizations: []Organization{
				{Name: "Acme Corp", Title: "Engineer"},
			},
		}
		assert.Equal(t, "Acme Corp", c.GetOrganization())
	})

	t.Run("returns title when no name", func(t *testing.T) {
		c := &Contact{
			Organizations: []Organization{
				{Title: "Freelance Developer"},
			},
		}
		assert.Equal(t, "Freelance Developer", c.GetOrganization())
	})

	t.Run("returns empty string when no organizations", func(t *testing.T) {
		c := &Contact{}
		assert.Equal(t, "", c.GetOrganization())
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
			assert.Equal(t, tt.expect, result)
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
			assert.Equal(t, tt.expect, result)
		})
	}
}

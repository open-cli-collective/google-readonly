package testutil

import (
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	calendarapi "github.com/open-cli-collective/google-readonly/internal/calendar"
	contactsapi "github.com/open-cli-collective/google-readonly/internal/contacts"
	driveapi "github.com/open-cli-collective/google-readonly/internal/drive"
	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
)

// Gmail fixtures

// SampleMessage returns a sample Gmail message for testing
func SampleMessage(id string) *gmailapi.Message {
	return &gmailapi.Message{
		ID:       id,
		ThreadID: "thread_" + id,
		Subject:  "Test Subject",
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Date:     "Mon, 1 Jan 2024 12:00:00 -0800",
		Snippet:  "This is a test message snippet...",
		Body:     "This is the full body of the test message.",
		Labels:   []string{"INBOX", "UNREAD"},
	}
}

// SampleMessages returns a slice of sample messages
func SampleMessages(count int) []*gmailapi.Message {
	messages := make([]*gmailapi.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = SampleMessage("msg_" + string(rune('a'+i)))
	}
	return messages
}

// SampleAttachment returns a sample attachment for testing
func SampleAttachment(filename string) *gmailapi.Attachment {
	return &gmailapi.Attachment{
		Filename:     filename,
		MimeType:     "application/pdf",
		Size:         1024,
		AttachmentID: "att_123",
		PartID:       "1",
		IsInline:     false,
	}
}

// SampleLabels returns sample Gmail labels for testing
func SampleLabels() []*gmail.Label {
	return []*gmail.Label{
		{Id: "INBOX", Name: "INBOX", Type: "system", MessagesTotal: 100, MessagesUnread: 5},
		{Id: "SENT", Name: "SENT", Type: "system", MessagesTotal: 50, MessagesUnread: 0},
		{Id: "Label_1", Name: "Work", Type: "user", MessagesTotal: 20, MessagesUnread: 2},
		{Id: "Label_2", Name: "Personal", Type: "user", MessagesTotal: 10, MessagesUnread: 1},
		{Id: "CATEGORY_SOCIAL", Name: "Social", Type: "system", MessagesTotal: 30, MessagesUnread: 10},
	}
}

// SampleProfile returns a sample Gmail profile for testing
func SampleProfile() *gmailapi.Profile {
	return &gmailapi.Profile{
		EmailAddress:  "user@example.com",
		MessagesTotal: 1000,
		ThreadsTotal:  500,
	}
}

// Calendar fixtures

// SampleCalendar returns a sample calendar for testing
func SampleCalendar(id string, primary bool) *calendar.CalendarListEntry {
	return &calendar.CalendarListEntry{
		Id:          id,
		Summary:     "Test Calendar",
		Description: "A test calendar",
		Primary:     primary,
		AccessRole:  "owner",
		TimeZone:    "America/Los_Angeles",
	}
}

// SampleCalendars returns sample calendars for testing
func SampleCalendars() []*calendar.CalendarListEntry {
	return []*calendar.CalendarListEntry{
		SampleCalendar("primary@example.com", true),
		SampleCalendar("work@example.com", false),
	}
}

// SampleEvent returns a sample calendar event for testing
func SampleEvent(id string) *calendar.Event {
	return &calendar.Event{
		Id:      id,
		Summary: "Test Meeting",
		Start: &calendar.EventDateTime{
			DateTime: "2024-01-15T10:00:00-08:00",
			TimeZone: "America/Los_Angeles",
		},
		End: &calendar.EventDateTime{
			DateTime: "2024-01-15T11:00:00-08:00",
			TimeZone: "America/Los_Angeles",
		},
		Location:    "Conference Room A",
		Description: "Discuss project progress",
		Organizer: &calendar.EventOrganizer{
			Email:       "organizer@example.com",
			DisplayName: "Meeting Organizer",
		},
		Attendees: []*calendar.EventAttendee{
			{Email: "alice@example.com", DisplayName: "Alice", ResponseStatus: "accepted"},
			{Email: "bob@example.com", DisplayName: "Bob", ResponseStatus: "tentative"},
		},
	}
}

// SampleParsedEvent returns a parsed calendar event for testing
func SampleParsedEvent(id string) *calendarapi.Event {
	return &calendarapi.Event{
		ID:      id,
		Summary: "Test Meeting",
		Start: &calendarapi.EventTime{
			DateTime: "2024-01-15T10:00:00-08:00",
			TimeZone: "America/Los_Angeles",
		},
		End: &calendarapi.EventTime{
			DateTime: "2024-01-15T11:00:00-08:00",
			TimeZone: "America/Los_Angeles",
		},
		AllDay:      false,
		Location:    "Conference Room A",
		Description: "Discuss project progress",
		Organizer: &calendarapi.Person{
			Email:       "organizer@example.com",
			DisplayName: "Meeting Organizer",
		},
		Attendees: []calendarapi.Person{
			{Email: "alice@example.com", DisplayName: "Alice", Status: "accepted"},
			{Email: "bob@example.com", DisplayName: "Bob", Status: "tentative"},
		},
	}
}

// Contacts fixtures

// SamplePerson returns a sample Google People API Person for testing
func SamplePerson(resourceName string) *people.Person {
	return &people.Person{
		ResourceName: resourceName,
		Names: []*people.Name{
			{DisplayName: "John Doe", GivenName: "John", FamilyName: "Doe"},
		},
		EmailAddresses: []*people.EmailAddress{
			{Value: "john@example.com", Type: "work"},
		},
		PhoneNumbers: []*people.PhoneNumber{
			{Value: "+1-555-123-4567", Type: "mobile"},
		},
		Organizations: []*people.Organization{
			{Name: "Acme Corp", Title: "Engineer"},
		},
	}
}

// SampleContact returns a parsed contact for testing
func SampleContact(resourceName string) *contactsapi.Contact {
	return &contactsapi.Contact{
		ResourceName: resourceName,
		Names: []contactsapi.Name{
			{DisplayName: "John Doe", GivenName: "John", FamilyName: "Doe"},
		},
		Emails: []contactsapi.Email{
			{Value: "john@example.com", Type: "work", Primary: true},
		},
		Phones: []contactsapi.Phone{
			{Value: "+1-555-123-4567", Type: "mobile"},
		},
		Organizations: []contactsapi.Organization{
			{Name: "Acme Corp", Title: "Engineer"},
		},
	}
}

// SampleContactGroup returns a sample contact group for testing
func SampleContactGroup(resourceName string) *contactsapi.ContactGroup {
	return &contactsapi.ContactGroup{
		ResourceName: resourceName,
		Name:         "Friends",
		GroupType:    "USER_CONTACT_GROUP",
		MemberCount:  5,
	}
}

// Drive fixtures

// SampleDriveFile returns a sample Drive file for testing
func SampleDriveFile(id string) *driveapi.File {
	return &driveapi.File{
		ID:           id,
		Name:         "test-document.pdf",
		MimeType:     "application/pdf",
		Size:         2048,
		CreatedTime:  time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
		ModifiedTime: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Parents:      []string{"root"},
		Owners:       []string{"owner@example.com"},
		WebViewLink:  "https://drive.google.com/file/d/" + id,
		Shared:       false,
	}
}

// SampleDriveFiles returns sample Drive files for testing
func SampleDriveFiles(count int) []*driveapi.File {
	files := make([]*driveapi.File, count)
	for i := 0; i < count; i++ {
		files[i] = SampleDriveFile("file_" + string(rune('a'+i)))
	}
	return files
}

// SampleGoogleDoc returns a Google Doc file for testing export scenarios
func SampleGoogleDoc(id string) *driveapi.File {
	return &driveapi.File{
		ID:           id,
		Name:         "My Document",
		MimeType:     driveapi.MimeTypeDocument,
		Size:         0, // Google Docs don't have size
		CreatedTime:  time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
		ModifiedTime: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Parents:      []string{"root"},
		Owners:       []string{"owner@example.com"},
		WebViewLink:  "https://docs.google.com/document/d/" + id,
		Shared:       false,
	}
}

// SampleSharedDrive returns a sample shared drive for testing
func SampleSharedDrive(id, name string) *driveapi.SharedDrive {
	return &driveapi.SharedDrive{
		ID:   id,
		Name: name,
	}
}

// SampleSharedDrives returns a set of sample shared drives for testing
func SampleSharedDrives() []*driveapi.SharedDrive {
	return []*driveapi.SharedDrive{
		{ID: "0ALengineering123", Name: "Engineering"},
		{ID: "0ALmarketing456", Name: "Marketing"},
		{ID: "0ALfinance789", Name: "Finance Team"},
	}
}

// SampleSharedDriveFile returns a file that belongs to a shared drive
func SampleSharedDriveFile(id, driveID string) *driveapi.File {
	return &driveapi.File{
		ID:           id,
		Name:         "shared-document.pdf",
		MimeType:     "application/pdf",
		Size:         4096,
		CreatedTime:  time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
		ModifiedTime: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Parents:      []string{driveID},
		Owners:       []string{},
		WebViewLink:  "https://drive.google.com/file/d/" + id,
		Shared:       true,
		DriveID:      driveID,
	}
}

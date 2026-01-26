// Package testutil provides test utilities including mock implementations
// of client interfaces for unit testing command handlers.
package testutil

import (
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	calendarapi "github.com/open-cli-collective/google-readonly/internal/calendar"
	contactsapi "github.com/open-cli-collective/google-readonly/internal/contacts"
	driveapi "github.com/open-cli-collective/google-readonly/internal/drive"
	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
)

// MockGmailClient is a configurable mock for GmailClientInterface.
// Set the function fields to control behavior in tests.
type MockGmailClient struct {
	GetMessageFunc               func(messageID string, includeBody bool) (*gmailapi.Message, error)
	SearchMessagesFunc           func(query string, maxResults int64) ([]*gmailapi.Message, int, error)
	GetThreadFunc                func(id string) ([]*gmailapi.Message, error)
	FetchLabelsFunc              func() error
	GetLabelNameFunc             func(labelID string) string
	GetLabelsFunc                func() []*gmail.Label
	GetAttachmentsFunc           func(messageID string) ([]*gmailapi.Attachment, error)
	DownloadAttachmentFunc       func(messageID, attachmentID string) ([]byte, error)
	DownloadInlineAttachmentFunc func(messageID, partID string) ([]byte, error)
	GetProfileFunc               func() (*gmailapi.Profile, error)
}

// Verify MockGmailClient implements GmailClientInterface
var _ gmailapi.GmailClientInterface = (*MockGmailClient)(nil)

func (m *MockGmailClient) GetMessage(messageID string, includeBody bool) (*gmailapi.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(messageID, includeBody)
	}
	return nil, nil
}

func (m *MockGmailClient) SearchMessages(query string, maxResults int64) ([]*gmailapi.Message, int, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(query, maxResults)
	}
	return nil, 0, nil
}

func (m *MockGmailClient) GetThread(id string) ([]*gmailapi.Message, error) {
	if m.GetThreadFunc != nil {
		return m.GetThreadFunc(id)
	}
	return nil, nil
}

func (m *MockGmailClient) FetchLabels() error {
	if m.FetchLabelsFunc != nil {
		return m.FetchLabelsFunc()
	}
	return nil
}

func (m *MockGmailClient) GetLabelName(labelID string) string {
	if m.GetLabelNameFunc != nil {
		return m.GetLabelNameFunc(labelID)
	}
	return labelID
}

func (m *MockGmailClient) GetLabels() []*gmail.Label {
	if m.GetLabelsFunc != nil {
		return m.GetLabelsFunc()
	}
	return nil
}

func (m *MockGmailClient) GetAttachments(messageID string) ([]*gmailapi.Attachment, error) {
	if m.GetAttachmentsFunc != nil {
		return m.GetAttachmentsFunc(messageID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadAttachment(messageID, attachmentID string) ([]byte, error) {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(messageID, attachmentID)
	}
	return nil, nil
}

func (m *MockGmailClient) DownloadInlineAttachment(messageID, partID string) ([]byte, error) {
	if m.DownloadInlineAttachmentFunc != nil {
		return m.DownloadInlineAttachmentFunc(messageID, partID)
	}
	return nil, nil
}

func (m *MockGmailClient) GetProfile() (*gmailapi.Profile, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc()
	}
	return nil, nil
}

// MockCalendarClient is a configurable mock for CalendarClientInterface.
type MockCalendarClient struct {
	ListCalendarsFunc func() ([]*calendar.CalendarListEntry, error)
	ListEventsFunc    func(calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error)
	GetEventFunc      func(calendarID, eventID string) (*calendar.Event, error)
}

// Verify MockCalendarClient implements CalendarClientInterface
var _ calendarapi.CalendarClientInterface = (*MockCalendarClient)(nil)

func (m *MockCalendarClient) ListCalendars() ([]*calendar.CalendarListEntry, error) {
	if m.ListCalendarsFunc != nil {
		return m.ListCalendarsFunc()
	}
	return nil, nil
}

func (m *MockCalendarClient) ListEvents(calendarID, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	if m.ListEventsFunc != nil {
		return m.ListEventsFunc(calendarID, timeMin, timeMax, maxResults)
	}
	return nil, nil
}

func (m *MockCalendarClient) GetEvent(calendarID, eventID string) (*calendar.Event, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(calendarID, eventID)
	}
	return nil, nil
}

// MockContactsClient is a configurable mock for ContactsClientInterface.
type MockContactsClient struct {
	ListContactsFunc      func(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error)
	SearchContactsFunc    func(query string, pageSize int64) (*people.SearchResponse, error)
	GetContactFunc        func(resourceName string) (*people.Person, error)
	ListContactGroupsFunc func(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error)
}

// Verify MockContactsClient implements ContactsClientInterface
var _ contactsapi.ContactsClientInterface = (*MockContactsClient)(nil)

func (m *MockContactsClient) ListContacts(pageToken string, pageSize int64) (*people.ListConnectionsResponse, error) {
	if m.ListContactsFunc != nil {
		return m.ListContactsFunc(pageToken, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) SearchContacts(query string, pageSize int64) (*people.SearchResponse, error) {
	if m.SearchContactsFunc != nil {
		return m.SearchContactsFunc(query, pageSize)
	}
	return nil, nil
}

func (m *MockContactsClient) GetContact(resourceName string) (*people.Person, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(resourceName)
	}
	return nil, nil
}

func (m *MockContactsClient) ListContactGroups(pageToken string, pageSize int64) (*people.ListContactGroupsResponse, error) {
	if m.ListContactGroupsFunc != nil {
		return m.ListContactGroupsFunc(pageToken, pageSize)
	}
	return nil, nil
}

// MockDriveClient is a configurable mock for DriveClientInterface.
type MockDriveClient struct {
	ListFilesFunc    func(query string, pageSize int64) ([]*driveapi.File, error)
	GetFileFunc      func(fileID string) (*driveapi.File, error)
	DownloadFileFunc func(fileID string) ([]byte, error)
	ExportFileFunc   func(fileID, mimeType string) ([]byte, error)
}

// Verify MockDriveClient implements DriveClientInterface
var _ driveapi.DriveClientInterface = (*MockDriveClient)(nil)

func (m *MockDriveClient) ListFiles(query string, pageSize int64) ([]*driveapi.File, error) {
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(query, pageSize)
	}
	return nil, nil
}

func (m *MockDriveClient) GetFile(fileID string) (*driveapi.File, error) {
	if m.GetFileFunc != nil {
		return m.GetFileFunc(fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) DownloadFile(fileID string) ([]byte, error) {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(fileID)
	}
	return nil, nil
}

func (m *MockDriveClient) ExportFile(fileID, mimeType string) ([]byte, error) {
	if m.ExportFileFunc != nil {
		return m.ExportFileFunc(fileID, mimeType)
	}
	return nil, nil
}

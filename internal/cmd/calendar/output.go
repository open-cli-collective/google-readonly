package calendar

import (
	"context"
	"fmt"

	calendarv3 "google.golang.org/api/calendar/v3"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// CalendarClient defines the interface for Calendar client operations used by calendar commands.
type CalendarClient interface {
	ListCalendars() ([]*calendarv3.CalendarListEntry, error)
	ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendarv3.Event, error)
	GetEvent(calendarID, eventID string) (*calendarv3.Event, error)
}

// ClientFactory is the function used to create Calendar clients.
// Override in tests to inject mocks.
var ClientFactory = func() (CalendarClient, error) {
	return calendar.NewClient(context.Background())
}

// newCalendarClient creates a new calendar client
func newCalendarClient() (CalendarClient, error) {
	return ClientFactory()
}

// printJSON outputs data as indented JSON
func printJSON(data any) error {
	return output.JSONStdout(data)
}

// printEvent prints a single event in text format
func printEvent(event *calendar.Event, showDescription bool) {
	fmt.Printf("ID: %s\n", event.ID)
	fmt.Printf("Summary: %s\n", event.Summary)
	fmt.Printf("When: %s\n", event.FormatTimeRange())

	if event.Location != "" {
		fmt.Printf("Location: %s\n", event.Location)
	}

	if event.HangoutLink != "" {
		fmt.Printf("Meet: %s\n", event.HangoutLink)
	}

	if event.Organizer != nil {
		if event.Organizer.DisplayName != "" {
			fmt.Printf("Organizer: %s <%s>\n", event.Organizer.DisplayName, event.Organizer.Email)
		} else {
			fmt.Printf("Organizer: %s\n", event.Organizer.Email)
		}
	}

	if len(event.Attendees) > 0 {
		fmt.Printf("Attendees: %d\n", len(event.Attendees))
		for _, a := range event.Attendees {
			status := ""
			if a.Status != "" {
				status = fmt.Sprintf(" (%s)", a.Status)
			}
			if a.DisplayName != "" {
				fmt.Printf("  - %s <%s>%s\n", a.DisplayName, a.Email, status)
			} else {
				fmt.Printf("  - %s%s\n", a.Email, status)
			}
		}
	}

	if showDescription && event.Description != "" {
		fmt.Println()
		fmt.Println("--- Description ---")
		fmt.Println(event.Description)
	}
}

// printEventSummary prints a brief event summary for list views
func printEventSummary(event *calendar.Event) {
	fmt.Printf("ID: %s\n", event.ID)
	fmt.Printf("Summary: %s\n", event.Summary)
	fmt.Printf("When: %s\n", event.FormatTimeRange())

	if event.Location != "" {
		fmt.Printf("Location: %s\n", event.Location)
	}

	if event.HangoutLink != "" {
		fmt.Printf("Meet: %s\n", event.HangoutLink)
	}

	fmt.Println("---")
}

// printCalendar prints a calendar entry
func printCalendar(cal *calendar.CalendarInfo) {
	primary := ""
	if cal.Primary {
		primary = " (primary)"
	}
	fmt.Printf("ID: %s%s\n", cal.ID, primary)
	fmt.Printf("Name: %s\n", cal.Summary)
	if cal.Description != "" {
		fmt.Printf("Description: %s\n", cal.Description)
	}
	fmt.Printf("Access: %s\n", cal.AccessRole)
	if cal.TimeZone != "" {
		fmt.Printf("Timezone: %s\n", cal.TimeZone)
	}
	fmt.Println("---")
}

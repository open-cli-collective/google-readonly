package calendar

import (
	"fmt"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

// EventListOptions configures how events are listed and displayed.
type EventListOptions struct {
	CalendarID   string
	TimeMin      string // RFC3339 format
	TimeMax      string // RFC3339 format
	MaxResults   int64
	JSONOutput   bool
	Header       string // Header message to print (empty to show count-based header)
	EmptyMessage string // Message when no events found
}

// listAndPrintEvents fetches events and prints them according to the options.
// This is a shared helper used by today, week, and events commands.
func listAndPrintEvents(client CalendarClient, opts EventListOptions) error {
	events, err := client.ListEvents(opts.CalendarID, opts.TimeMin, opts.TimeMax, opts.MaxResults)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		if opts.EmptyMessage != "" {
			fmt.Println(opts.EmptyMessage)
		} else {
			fmt.Println("No events found.")
		}
		return nil
	}

	// Convert to our format
	parsedEvents := make([]*calendar.Event, len(events))
	for i, e := range events {
		parsedEvents[i] = calendar.ParseEvent(e)
	}

	if opts.JSONOutput {
		return printJSON(parsedEvents)
	}

	// Print header
	if opts.Header != "" {
		fmt.Printf("%s\n\n", opts.Header)
	} else {
		fmt.Printf("Found %d event(s):\n\n", len(events))
	}

	for _, event := range parsedEvents {
		printEventSummary(event)
	}

	return nil
}

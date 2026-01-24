package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

var (
	eventsCalendarID string
	eventsMaxResults int64
	eventsFrom       string
	eventsTo         string
	eventsJSONOutput bool
)

func newEventsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events [calendar-id]",
		Short: "List calendar events",
		Long: `List events from a calendar.

By default, shows upcoming events from the primary calendar.
Use --from and --to flags to specify a date range.

Date format: YYYY-MM-DD (e.g., 2026-01-24)

Examples:
  gro calendar events
  gro cal events --max 20
  gro cal events --from 2026-01-01 --to 2026-01-31
  gro calendar events work@group.calendar.google.com --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: runEvents,
	}

	cmd.Flags().StringVarP(&eventsCalendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().Int64VarP(&eventsMaxResults, "max", "m", 10, "Maximum number of events to return")
	cmd.Flags().StringVar(&eventsFrom, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&eventsTo, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().BoolVarP(&eventsJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runEvents(cmd *cobra.Command, args []string) error {
	calendarID := eventsCalendarID
	if len(args) > 0 {
		calendarID = args[0]
	}

	client, err := newCalendarClient()
	if err != nil {
		return err
	}

	// Parse date range
	var timeMin, timeMax string

	if eventsFrom != "" {
		t, err := parseDate(eventsFrom)
		if err != nil {
			return fmt.Errorf("invalid --from date: %w", err)
		}
		timeMin = t.Format(time.RFC3339)
	} else {
		// Default to now
		timeMin = time.Now().Format(time.RFC3339)
	}

	if eventsTo != "" {
		t, err := parseDate(eventsTo)
		if err != nil {
			return fmt.Errorf("invalid --to date: %w", err)
		}
		// Set to end of day
		timeMax = endOfDay(t).Format(time.RFC3339)
	}

	events, err := client.ListEvents(calendarID, timeMin, timeMax, eventsMaxResults)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println("No events found.")
		return nil
	}

	// Convert to our format
	parsedEvents := make([]*calendar.Event, len(events))
	for i, e := range events {
		parsedEvents[i] = calendar.ParseEvent(e)
	}

	if eventsJSONOutput {
		return printJSON(parsedEvents)
	}

	fmt.Printf("Found %d event(s):\n\n", len(events))
	for _, event := range parsedEvents {
		printEventSummary(event)
	}

	return nil
}

package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

func newEventsCommand() *cobra.Command {
	var (
		calendarID string
		maxResults int64
		from       string
		to         string
		jsonOutput bool
	)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			calID := calendarID
			if len(args) > 0 {
				calID = args[0]
			}

			client, err := newCalendarClient()
			if err != nil {
				return fmt.Errorf("failed to create Calendar client: %w", err)
			}

			// Parse date range
			var timeMin, timeMax string

			if from != "" {
				t, err := parseDate(from)
				if err != nil {
					return fmt.Errorf("invalid --from date: %w", err)
				}
				timeMin = t.Format(time.RFC3339)
			} else {
				// Default to now
				timeMin = time.Now().Format(time.RFC3339)
			}

			if to != "" {
				t, err := parseDate(to)
				if err != nil {
					return fmt.Errorf("invalid --to date: %w", err)
				}
				// Set to end of day
				timeMax = endOfDay(t).Format(time.RFC3339)
			}

			events, err := client.ListEvents(calID, timeMin, timeMax, maxResults)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
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

			if jsonOutput {
				return printJSON(parsedEvents)
			}

			fmt.Printf("Found %d event(s):\n\n", len(events))
			for _, event := range parsedEvents {
				printEventSummary(event)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().Int64VarP(&maxResults, "max", "m", 10, "Maximum number of events to return")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

func newWeekCommand() *cobra.Command {
	var (
		calendarID string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "week",
		Short: "Show this week's events",
		Long: `Show all events for the current week (Monday to Sunday).

This is a shortcut for: gro calendar events --from <monday> --to <sunday>

Examples:
  gro calendar week
  gro cal week --json
  gro cal week --calendar work@group.calendar.google.com`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return fmt.Errorf("failed to create Calendar client: %w", err)
			}

			now := time.Now()
			startOfWeek, endOfWeek := weekBounds(now)

			timeMin := startOfWeek.Format(time.RFC3339)
			timeMax := endOfWeek.Format(time.RFC3339)

			events, err := client.ListEvents(calendarID, timeMin, timeMax, 100)
			if err != nil {
				return fmt.Errorf("failed to list week's events: %w", err)
			}

			if len(events) == 0 {
				fmt.Println("No events this week.")
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

			fmt.Printf("This week's events (%s - %s):\n\n",
				startOfWeek.Format("Mon, Jan 2"),
				endOfWeek.Format("Mon, Jan 2, 2006"))
			for _, event := range parsedEvents {
				printEventSummary(event)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

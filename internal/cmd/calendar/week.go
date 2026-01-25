package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
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

			return listAndPrintEvents(client, EventListOptions{
				CalendarID: calendarID,
				TimeMin:    startOfWeek.Format(time.RFC3339),
				TimeMax:    endOfWeek.Format(time.RFC3339),
				MaxResults: 100,
				JSONOutput: jsonOutput,
				Header: fmt.Sprintf("This week's events (%s - %s):",
					startOfWeek.Format("Mon, Jan 2"),
					endOfWeek.Format("Mon, Jan 2, 2006")),
				EmptyMessage: "No events this week.",
			})
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

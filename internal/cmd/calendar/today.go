package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newTodayCommand() *cobra.Command {
	var (
		calendarID string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "today",
		Short: "Show today's events",
		Long: `Show all events for today.

This is a shortcut for: gro calendar events --from <today> --to <today>

Examples:
  gro calendar today
  gro cal today --json
  gro cal today --calendar work@group.calendar.google.com`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := newCalendarClient()
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}

			now := time.Now()
			startOfDay, endOfDayTime := todayBounds(now)

			return listAndPrintEvents(client, EventListOptions{
				CalendarID:   calendarID,
				TimeMin:      startOfDay.Format(time.RFC3339),
				TimeMax:      endOfDayTime.Format(time.RFC3339),
				MaxResults:   50,
				JSONOutput:   jsonOutput,
				Header:       fmt.Sprintf("Today's events (%s):", now.Format("Mon, Jan 2, 2006")),
				EmptyMessage: "No events today.",
			})
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

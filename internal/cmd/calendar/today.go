package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

var (
	todayCalendarID string
	todayJSONOutput bool
)

func newTodayCommand() *cobra.Command {
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
		RunE: runToday,
	}

	cmd.Flags().StringVarP(&todayCalendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().BoolVarP(&todayJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runToday(cmd *cobra.Command, args []string) error {
	client, err := newCalendarClient()
	if err != nil {
		return err
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24*time.Hour - time.Second)

	timeMin := startOfDay.Format(time.RFC3339)
	timeMax := endOfDay.Format(time.RFC3339)

	events, err := client.ListEvents(todayCalendarID, timeMin, timeMax, 50)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		fmt.Println("No events today.")
		return nil
	}

	// Convert to our format
	parsedEvents := make([]*calendar.Event, len(events))
	for i, e := range events {
		parsedEvents[i] = calendar.ParseEvent(e)
	}

	if todayJSONOutput {
		return printJSON(parsedEvents)
	}

	fmt.Printf("Today's events (%s):\n\n", now.Format("Mon, Jan 2, 2006"))
	for _, event := range parsedEvents {
		printEventSummary(event)
	}

	return nil
}

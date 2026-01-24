package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

var (
	weekCalendarID string
	weekJSONOutput bool
)

func newWeekCommand() *cobra.Command {
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
		RunE: runWeek,
	}

	cmd.Flags().StringVarP(&weekCalendarID, "calendar", "c", "primary", "Calendar ID to query")
	cmd.Flags().BoolVarP(&weekJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runWeek(cmd *cobra.Command, args []string) error {
	client, err := newCalendarClient()
	if err != nil {
		return err
	}

	now := time.Now()

	// Find Monday of this week
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday becomes 7
	}
	monday := now.AddDate(0, 0, -weekday+1)
	startOfWeek := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())

	// Find Sunday of this week
	sunday := startOfWeek.AddDate(0, 0, 6)
	endOfWeek := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, now.Location())

	timeMin := startOfWeek.Format(time.RFC3339)
	timeMax := endOfWeek.Format(time.RFC3339)

	events, err := client.ListEvents(weekCalendarID, timeMin, timeMax, 100)
	if err != nil {
		return err
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

	if weekJSONOutput {
		return printJSON(parsedEvents)
	}

	fmt.Printf("This week's events (%s - %s):\n\n",
		startOfWeek.Format("Mon, Jan 2"),
		endOfWeek.Format("Mon, Jan 2, 2006"))
	for _, event := range parsedEvents {
		printEventSummary(event)
	}

	return nil
}

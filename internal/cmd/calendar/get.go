package calendar

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

var (
	getCalendarID string
	getJSONOutput bool
)

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <event-id>",
		Short: "Get event details",
		Long: `Get the full details of a calendar event.

Shows summary, time, location, description, attendees, and meeting links.

Examples:
  gro calendar get abc123xyz
  gro cal get abc123xyz --json
  gro cal get abc123xyz --calendar work@group.calendar.google.com`,
		Args: cobra.ExactArgs(1),
		RunE: runGet,
	}

	cmd.Flags().StringVarP(&getCalendarID, "calendar", "c", "primary", "Calendar ID containing the event")
	cmd.Flags().BoolVarP(&getJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	eventID := args[0]

	client, err := newCalendarClient()
	if err != nil {
		return err
	}

	event, err := client.GetEvent(getCalendarID, eventID)
	if err != nil {
		return err
	}

	parsedEvent := calendar.ParseEvent(event)

	if getJSONOutput {
		return printJSON(parsedEvent)
	}

	printEvent(parsedEvent, true)
	return nil
}

package calendar

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

func newGetCommand() *cobra.Command {
	var (
		calendarID string
		jsonOutput bool
	)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]

			client, err := newCalendarClient()
			if err != nil {
				return fmt.Errorf("failed to create Calendar client: %w", err)
			}

			event, err := client.GetEvent(calendarID, eventID)
			if err != nil {
				return fmt.Errorf("failed to get event: %w", err)
			}

			parsedEvent := calendar.ParseEvent(event)

			if jsonOutput {
				return printJSON(parsedEvent)
			}

			printEvent(parsedEvent, true)
			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID containing the event")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

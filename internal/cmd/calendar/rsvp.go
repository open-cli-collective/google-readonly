package calendar

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// validResponses maps user-friendly input to Google Calendar API response values.
var validResponses = map[string]string{
	"accept":    "accepted",
	"decline":   "declined",
	"tentative": "tentative",
}

func newRSVPCommand() *cobra.Command {
	var (
		calendarID string
		jsonOutput bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "rsvp <event-id> <accept|decline|tentative>",
		Short: "Update your RSVP status on an event",
		Long: `Update your RSVP response for a calendar event.

Valid responses: accept, decline, tentative

Examples:
  gro calendar rsvp abc123 accept
  gro cal rsvp abc123 decline --dry-run
  gro cal rsvp abc123 tentative --calendar work@group.calendar.google.com --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]
			input := strings.ToLower(args[1])

			apiResponse, ok := validResponses[input]
			if !ok {
				return fmt.Errorf("invalid response %q; must be accept, decline, or tentative", input)
			}

			if dryRun {
				result := map[string]any{
					"action":   fmt.Sprintf("RSVP '%s' to event", apiResponse),
					"eventId":  eventID,
					"response": apiResponse,
					"dryRun":   true,
				}
				if jsonOutput {
					return printJSON(result)
				}
				fmt.Printf("[dry-run] Would RSVP '%s' to event %s.\n", apiResponse, eventID)
				return nil
			}

			client, err := newCalendarClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}

			if err := client.RSVPEvent(cmd.Context(), calendarID, eventID, apiResponse); err != nil {
				return fmt.Errorf("updating RSVP: %w", err)
			}

			result := map[string]any{
				"action":   fmt.Sprintf("RSVP'd '%s' to event", apiResponse),
				"eventId":  eventID,
				"response": apiResponse,
				"dryRun":   false,
			}
			if jsonOutput {
				return printJSON(result)
			}
			fmt.Printf("RSVP'd '%s' to event %s.\n", apiResponse, eventID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID containing the event")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")

	return cmd
}

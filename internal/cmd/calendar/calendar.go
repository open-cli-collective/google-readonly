// Package calendar implements the gro calendar command and subcommands.
package calendar

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the calendar command with all subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Aliases: []string{"cal"},
		Short:   "Google Calendar commands",
		Long: `Access to Google Calendar events and calendars.

This command group provides Calendar functionality:
- list: List all calendars
- events: List upcoming events
- get: View a single event's details
- today: Show today's events
- week: Show this week's events
- rsvp: Update your RSVP status on an event
- color: Set event color

Examples:
  gro calendar list
  gro cal events --max 20
  gro cal today
  gro calendar get <event-id>
  gro cal rsvp <event-id> accept
  gro cal color <event-id> tomato`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newEventsCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newTodayCommand())
	cmd.AddCommand(newWeekCommand())
	cmd.AddCommand(newRSVPCommand())
	cmd.AddCommand(newColorCommand())

	return cmd
}

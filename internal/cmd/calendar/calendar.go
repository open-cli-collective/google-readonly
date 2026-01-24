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
		Long: `Read-only access to Google Calendar events and calendars.

This command group provides Calendar functionality:
- list: List all calendars
- events: List upcoming events
- get: View a single event's details
- today: Show today's events
- week: Show this week's events

Examples:
  gro calendar list
  gro cal events --max 20
  gro cal today --json
  gro calendar get <event-id>`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newEventsCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newTodayCommand())
	cmd.AddCommand(newWeekCommand())

	return cmd
}

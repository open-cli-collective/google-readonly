package calendar

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

func newListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all calendars",
		Long: `List all calendars the user has access to.

Shows primary calendar, shared calendars, and subscribed calendars.

Examples:
  gro calendar list
  gro cal list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCalendarClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}

			calendars, err := client.ListCalendars()
			if err != nil {
				return fmt.Errorf("listing calendars: %w", err)
			}

			if len(calendars) == 0 {
				fmt.Println("No calendars found.")
				return nil
			}

			// Convert to our format
			calInfos := make([]*calendar.CalendarInfo, len(calendars))
			for i, c := range calendars {
				calInfos[i] = calendar.ParseCalendar(c)
			}

			if jsonOutput {
				return printJSON(calInfos)
			}

			fmt.Printf("Found %d calendar(s):\n\n", len(calendars))
			for _, cal := range calInfos {
				printCalendar(cal)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

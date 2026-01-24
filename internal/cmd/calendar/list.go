package calendar

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/calendar"
)

var (
	listJSONOutput bool
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all calendars",
		Long: `List all calendars the user has access to.

Shows primary calendar, shared calendars, and subscribed calendars.

Examples:
  gro calendar list
  gro cal list --json`,
		Args: cobra.NoArgs,
		RunE: runList,
	}

	cmd.Flags().BoolVarP(&listJSONOutput, "json", "j", false, "Output as JSON")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := newCalendarClient()
	if err != nil {
		return err
	}

	calendars, err := client.ListCalendars()
	if err != nil {
		return err
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

	if listJSONOutput {
		return printJSON(calInfos)
	}

	fmt.Printf("Found %d calendar(s):\n\n", len(calendars))
	for _, cal := range calInfos {
		printCalendar(cal)
	}

	return nil
}

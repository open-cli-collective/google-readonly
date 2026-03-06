package calendar

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// eventColors maps Google Calendar event color names to their IDs (1-11).
var eventColors = map[string]string{
	"lavender":  "1",
	"sage":      "2",
	"grape":     "3",
	"flamingo":  "4",
	"banana":    "5",
	"tangerine": "6",
	"peacock":   "7",
	"graphite":  "8",
	"blueberry": "9",
	"basil":     "10",
	"tomato":    "11",
}

// colorIDToName provides reverse lookup from color ID to name.
var colorIDToName = func() map[string]string {
	m := make(map[string]string, len(eventColors))
	for name, id := range eventColors {
		m[id] = name
	}
	return m
}()

func newColorCommand() *cobra.Command {
	var (
		calendarID string
		jsonOutput bool
		dryRun     bool
	)

	colorNames := make([]string, 0, len(eventColors))
	for name := range eventColors {
		colorNames = append(colorNames, name)
	}
	sort.Strings(colorNames)

	cmd := &cobra.Command{
		Use:   "color <event-id> <color>",
		Short: "Set event color",
		Long: fmt.Sprintf(`Set the color of a calendar event.

Accepts a color name or ID (1-11).

Color names: %s

Examples:
  gro calendar color abc123 tomato
  gro cal color abc123 7
  gro cal color abc123 sage --dry-run --json`, strings.Join(colorNames, ", ")),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]
			colorInput := strings.ToLower(args[1])

			// Resolve color input to ID and name
			colorID, colorName, err := resolveColor(colorInput)
			if err != nil {
				return err
			}

			if dryRun {
				result := map[string]any{
					"action":    fmt.Sprintf("set event color to %s", colorName),
					"eventId":   eventID,
					"colorId":   colorID,
					"colorName": colorName,
					"dryRun":    true,
				}
				if jsonOutput {
					return printJSON(result)
				}
				fmt.Printf("[dry-run] Would set event %s color to %s.\n", eventID, colorName)
				return nil
			}

			client, err := newCalendarClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}

			if err := client.SetEventColor(cmd.Context(), calendarID, eventID, colorID); err != nil {
				return fmt.Errorf("setting event color: %w", err)
			}

			result := map[string]any{
				"action":    fmt.Sprintf("set event color to %s", colorName),
				"eventId":   eventID,
				"colorId":   colorID,
				"colorName": colorName,
				"dryRun":    false,
			}
			if jsonOutput {
				return printJSON(result)
			}
			fmt.Printf("Set event %s color to %s.\n", eventID, colorName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID containing the event")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")

	return cmd
}

// resolveColor resolves a color name or numeric ID to (colorID, colorName).
func resolveColor(input string) (string, string, error) {
	// Try as color name first
	if id, ok := eventColors[input]; ok {
		return id, input, nil
	}

	// Try as numeric ID
	n, err := strconv.Atoi(input)
	if err == nil && n >= 1 && n <= 11 {
		id := strconv.Itoa(n)
		name := colorIDToName[id]
		return id, name, nil
	}

	colorNames := make([]string, 0, len(eventColors))
	for name := range eventColors {
		colorNames = append(colorNames, name)
	}
	sort.Strings(colorNames)
	return "", "", fmt.Errorf("invalid color %q; valid colors: %s (or 1-11)", input, strings.Join(colorNames, ", "))
}

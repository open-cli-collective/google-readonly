package mail

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	gmailapi "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-readonly/internal/format"
)

// Label represents a Gmail label for output
type Label struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	MessagesTotal  int64  `json:"messagesTotal,omitempty"`
	MessagesUnread int64  `json:"messagesUnread,omitempty"`
}

func newLabelsCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "labels",
		Short: "List all labels",
		Long: `List all Gmail labels including user labels and system categories.

Shows label name, type (system/user/category), and message counts.

Examples:
  gro mail labels
  gro mail labels --json`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := newGmailClient()
			if err != nil {
				return fmt.Errorf("failed to create Gmail client: %w", err)
			}

			if err := client.FetchLabels(); err != nil {
				return fmt.Errorf("failed to fetch labels: %w", err)
			}

			gmailLabels := client.GetLabels()
			if len(gmailLabels) == 0 {
				fmt.Println("No labels found.")
				return nil
			}

			// Convert to our Label type and categorize
			labels := make([]Label, 0, len(gmailLabels))
			for _, gl := range gmailLabels {
				label := Label{
					ID:             gl.Id,
					Name:           gl.Name,
					Type:           getLabelType(gl),
					MessagesTotal:  gl.MessagesTotal,
					MessagesUnread: gl.MessagesUnread,
				}
				labels = append(labels, label)
			}

			// Sort by type then name
			sort.Slice(labels, func(i, j int) bool {
				if labels[i].Type != labels[j].Type {
					return labelTypePriority(labels[i].Type) < labelTypePriority(labels[j].Type)
				}
				return strings.ToLower(labels[i].Name) < strings.ToLower(labels[j].Name)
			})

			if jsonOutput {
				return printJSON(labels)
			}

			// Text output
			fmt.Printf("%-30s %-10s %8s %8s\n", "NAME", "TYPE", "TOTAL", "UNREAD")
			fmt.Println(strings.Repeat("-", 60))
			for _, label := range labels {
				fmt.Printf("%-30s %-10s %8d %8d\n",
					format.Truncate(label.Name, 30),
					label.Type,
					label.MessagesTotal,
					label.MessagesUnread)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

func getLabelType(gl *gmailapi.Label) string {
	// Check for categories
	if strings.HasPrefix(gl.Id, "CATEGORY_") {
		return "category"
	}

	// System labels have Type = "system"
	if gl.Type == "system" {
		return "system"
	}

	return "user"
}

func labelTypePriority(t string) int {
	switch t {
	case "user":
		return 0
	case "category":
		return 1
	case "system":
		return 2
	default:
		return 3
	}
}

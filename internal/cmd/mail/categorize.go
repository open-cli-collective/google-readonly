package mail

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/bulk"
)

// categoryLabels maps user-friendly category names to Gmail label IDs.
var categoryLabels = map[string]string{
	"personal":   "CATEGORY_PERSONAL",
	"social":     "CATEGORY_SOCIAL",
	"promotions": "CATEGORY_PROMOTIONS",
	"updates":    "CATEGORY_UPDATES",
	"forums":     "CATEGORY_FORUMS",
}

// allCategoryLabelIDs returns all category label IDs for removal when recategorizing.
func allCategoryLabelIDs() []string {
	ids := make([]string, 0, len(categoryLabels))
	for _, id := range categoryLabels {
		ids = append(ids, id)
	}
	return ids
}

func newCategorizeCommand() *cobra.Command {
	var (
		jsonOutput bool
		dryRun     bool
		stdin      bool
		query      string
	)

	validCategories := make([]string, 0, len(categoryLabels))
	for name := range categoryLabels {
		validCategories = append(validCategories, name)
	}
	sort.Strings(validCategories)

	cmd := &cobra.Command{
		Use:   "categorize <category> [message-ids...]",
		Short: "Recategorize messages",
		Long: fmt.Sprintf(`Move Gmail messages to a different category tab.

This removes all existing category labels and adds the target category.

Valid categories: %s

Examples:
  gro mail categorize promotions msg123
  gro mail categorize social --query "from:linkedin"
  gro mail categorize promotions --query "category:updates from:noreply" --dry-run`, strings.Join(validCategories, ", ")),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			category := strings.ToLower(args[0])
			messageArgs := args[1:]

			targetLabel, ok := categoryLabels[category]
			if !ok {
				return fmt.Errorf("invalid category %q; valid categories: %s", category, strings.Join(validCategories, ", "))
			}

			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ctx := cmd.Context()
			ids, err := bulk.ResolveIDs(bulk.Config{
				Args:  messageArgs,
				Stdin: stdin,
				Query: query,
			}, func(q string) ([]string, error) {
				return client.SearchMessageIDs(ctx, q, 0)
			})
			if err != nil {
				return err
			}

			result := &bulk.Result{
				Action:  fmt.Sprintf("recategorized to %s", category),
				IDs:     ids,
				Count:   len(ids),
				DryRun:  dryRun,
				Details: map[string]string{"category": category, "labelId": targetLabel},
			}

			if dryRun {
				result.Action = fmt.Sprintf("recategorize to %s", category)
				return result.Print(jsonOutput)
			}

			// Remove all category labels and add the target one
			removeLabels := allCategoryLabelIDs()
			if err := client.ModifyMessages(ctx, ids, []string{targetLabel}, removeLabels); err != nil {
				return fmt.Errorf("recategorizing messages: %w", err)
			}

			return result.Print(jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")

	return cmd
}

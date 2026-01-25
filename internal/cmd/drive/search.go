package drive

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	var (
		maxResults int64
		nameOnly   bool
		fileType   string
		owner      string
		modAfter   string
		modBefore  string
		inFolder   string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for files",
		Long: `Search for files in Google Drive by content, name, type, owner, or date.

Examples:
  gro drive search "quarterly report"           # Full-text search
  gro drive search --name "budget"              # Search by filename only
  gro drive search --type spreadsheet           # Filter by type
  gro drive search --owner me                   # Files you own
  gro drive search --owner john@example.com     # Files owned by someone
  gro drive search --modified-after 2024-01-01  # Modified after date
  gro drive search --in-folder <folder-id>      # Search within folder
  gro drive search "report" --type document --max 20 --json

File types: document, spreadsheet, presentation, folder, pdf, image, video, audio`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("failed to create Drive client: %w", err)
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			searchQuery, err := buildSearchQuery(query, nameOnly, fileType, owner, modAfter, modBefore, inFolder)
			if err != nil {
				return fmt.Errorf("failed to build search query: %w", err)
			}

			files, err := client.ListFiles(searchQuery, maxResults)
			if err != nil {
				return fmt.Errorf("failed to search files: %w", err)
			}

			if len(files) == 0 {
				if query != "" {
					fmt.Printf("No files found matching \"%s\".\n", query)
				} else {
					fmt.Println("No files found.")
				}
				return nil
			}

			if jsonOutput {
				return printJSON(files)
			}

			if query != "" {
				fmt.Printf("Found %d file(s) matching \"%s\":\n\n", len(files), query)
			} else {
				fmt.Printf("Found %d file(s):\n\n", len(files))
			}
			printFileTable(files)
			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().BoolVarP(&nameOnly, "name", "n", false, "Search filename only (not content)")
	cmd.Flags().StringVarP(&fileType, "type", "t", "", "Filter by file type")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter by owner (\"me\" or email address)")
	cmd.Flags().StringVar(&modAfter, "modified-after", "", "Modified after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&modBefore, "modified-before", "", "Modified before date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&inFolder, "in-folder", "", "Search within specific folder")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

// buildSearchQuery constructs a Drive API query string for searching files
func buildSearchQuery(query string, nameOnly bool, fileType, owner, modAfter, modBefore, inFolder string) (string, error) {
	parts := []string{"trashed = false"}

	// Text search
	if query != "" {
		escaped := escapeQueryString(query)
		if nameOnly {
			parts = append(parts, fmt.Sprintf("name contains '%s'", escaped))
		} else {
			parts = append(parts, fmt.Sprintf("fullText contains '%s'", escaped))
		}
	}

	// Type filter
	if fileType != "" {
		filter, err := getMimeTypeFilter(fileType)
		if err != nil {
			return "", err
		}
		parts = append(parts, filter)
	}

	// Owner filter
	if owner != "" {
		parts = append(parts, fmt.Sprintf("'%s' in owners", owner))
	}

	// Date filters
	if modAfter != "" {
		// Drive API requires RFC3339 format
		parts = append(parts, fmt.Sprintf("modifiedTime > '%sT00:00:00'", modAfter))
	}
	if modBefore != "" {
		parts = append(parts, fmt.Sprintf("modifiedTime < '%sT23:59:59'", modBefore))
	}

	// Folder scope
	if inFolder != "" {
		parts = append(parts, fmt.Sprintf("'%s' in parents", inFolder))
	}

	return strings.Join(parts, " and "), nil
}

// escapeQueryString escapes special characters in search queries
func escapeQueryString(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "\\'")
}

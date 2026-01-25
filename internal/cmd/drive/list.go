package drive

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/drive"
)

func newListCommand() *cobra.Command {
	var (
		maxResults int64
		fileType   string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list [folder-id]",
		Short: "List files in Drive",
		Long: `List files in Google Drive root or a specific folder.

Examples:
  gro drive list                    # List files in root
  gro drive list <folder-id>        # List files in specific folder
  gro drive list --type document    # Filter by file type
  gro drive list --max 50           # Limit results
  gro drive list --json             # Output as JSON

File types: document, spreadsheet, presentation, folder, pdf, image, video, audio`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("failed to create Drive client: %w", err)
			}

			folderID := ""
			if len(args) > 0 {
				folderID = args[0]
			}

			query, err := buildListQuery(folderID, fileType)
			if err != nil {
				return fmt.Errorf("failed to build query: %w", err)
			}

			files, err := client.ListFiles(query, maxResults)
			if err != nil {
				return fmt.Errorf("failed to list files: %w", err)
			}

			if len(files) == 0 {
				fmt.Println("No files found.")
				return nil
			}

			if jsonOutput {
				return printJSON(files)
			}

			printFileTable(files)
			return nil
		},
	}

	cmd.Flags().Int64VarP(&maxResults, "max", "m", 25, "Maximum number of results to return")
	cmd.Flags().StringVarP(&fileType, "type", "t", "", "Filter by file type")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

// buildListQuery constructs a Drive API query string for listing files
func buildListQuery(folderID, fileType string) (string, error) {
	parts := []string{"trashed = false"}

	if folderID != "" {
		parts = append(parts, fmt.Sprintf("'%s' in parents", folderID))
	} else {
		parts = append(parts, "'root' in parents")
	}

	if fileType != "" {
		filter, err := getMimeTypeFilter(fileType)
		if err != nil {
			return "", err
		}
		parts = append(parts, filter)
	}

	return strings.Join(parts, " and "), nil
}

// getMimeTypeFilter returns the Drive API query filter for a file type
func getMimeTypeFilter(fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "document", "doc":
		return fmt.Sprintf("mimeType = '%s'", drive.MimeTypeDocument), nil
	case "spreadsheet", "sheet":
		return fmt.Sprintf("mimeType = '%s'", drive.MimeTypeSpreadsheet), nil
	case "presentation", "slides":
		return fmt.Sprintf("mimeType = '%s'", drive.MimeTypePresentation), nil
	case "folder":
		return fmt.Sprintf("mimeType = '%s'", drive.MimeTypeFolder), nil
	case "pdf":
		return "mimeType = 'application/pdf'", nil
	case "image":
		return "mimeType contains 'image/'", nil
	case "video":
		return "mimeType contains 'video/'", nil
	case "audio":
		return "mimeType contains 'audio/'", nil
	default:
		return "", fmt.Errorf("unknown file type: %s (valid types: document, spreadsheet, presentation, folder, pdf, image, video, audio)", fileType)
	}
}

// printFileTable prints files in a formatted table
func printFileTable(files []*drive.File) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tSIZE\tMODIFIED")

	for _, f := range files {
		size := "-"
		if f.Size > 0 {
			size = formatSize(f.Size)
		}

		modified := "-"
		if !f.ModifiedTime.IsZero() {
			modified = f.ModifiedTime.Format("2006-01-02")
		}

		typeName := drive.GetTypeName(f.MimeType)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			f.ID, f.Name, typeName, size, modified)
	}

	_ = w.Flush()
}

// formatSize formats a byte size into a human-readable string
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

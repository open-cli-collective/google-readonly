package drive

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/format"
)

func newListCommand() *cobra.Command {
	var (
		maxResults int64
		fileType   string
		jsonOutput bool
		myDrive    bool
		driveFlag  string
	)

	cmd := &cobra.Command{
		Use:   "list [folder-id]",
		Short: "List files in Drive",
		Long: `List files in Google Drive root or a specific folder.

By default, lists files in My Drive root. Use --drive to list files in a
specific shared drive's root.

Examples:
  gro drive list                        # List files in My Drive root
  gro drive list <folder-id>            # List files in specific folder
  gro drive list --drive "Engineering"  # List files in shared drive root
  gro drive list --type document        # Filter by file type
  gro drive list --max 50               # Limit results
  gro drive list --json                 # Output as JSON

File types: document, spreadsheet, presentation, folder, pdf, image, video, audio`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Validate mutually exclusive flags
			if myDrive && driveFlag != "" {
				return fmt.Errorf("--my-drive and --drive are mutually exclusive")
			}

			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}

			folderID := ""
			if len(args) > 0 {
				folderID = args[0]
			}

			// Resolve drive scope for listing
			scope, err := resolveDriveScopeForList(client, myDrive, driveFlag, folderID)
			if err != nil {
				return fmt.Errorf("resolving drive scope: %w", err)
			}

			query, err := buildListQueryWithScope(folderID, fileType, scope)
			if err != nil {
				return fmt.Errorf("building query: %w", err)
			}

			files, err := client.ListFilesWithScope(query, maxResults, scope)
			if err != nil {
				return fmt.Errorf("listing files: %w", err)
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
	cmd.Flags().BoolVar(&myDrive, "my-drive", false, "Limit to My Drive only")
	cmd.Flags().StringVar(&driveFlag, "drive", "", "List files in specific shared drive (name or ID)")

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

// buildListQueryWithScope constructs a Drive API query string with scope awareness
func buildListQueryWithScope(folderID, fileType string, scope drive.DriveScope) (string, error) {
	parts := []string{"trashed = false"}

	// For shared drives, if no folder specified, we don't add 'root' in parents
	// because the root is the drive itself
	if folderID != "" {
		parts = append(parts, fmt.Sprintf("'%s' in parents", folderID))
	} else if scope.DriveID == "" {
		// Only add 'root' for My Drive listings
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

// resolveDriveScopeForList resolves the scope for list operations
// List has slightly different behavior - defaults to My Drive root if no flags
func resolveDriveScopeForList(client drive.DriveClientInterface, myDrive bool, driveFlag, folderID string) (drive.DriveScope, error) {
	// If a folder ID is provided, we need to support all drives to access it
	if folderID != "" && !myDrive && driveFlag == "" {
		return drive.DriveScope{AllDrives: true}, nil
	}

	// Otherwise use the standard resolution
	return resolveDriveScope(client, myDrive, driveFlag)
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

// printFileTable prints files in a formatted table.
// Write errors to stdout are intentionally ignored as they indicate
// the output stream is closed/broken and there's nothing useful to do.
func printFileTable(files []*drive.File) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tSIZE\tMODIFIED")

	for _, f := range files {
		size := "-"
		if f.Size > 0 {
			size = format.Size(f.Size)
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

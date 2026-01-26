package drive

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/drive"
	"github.com/open-cli-collective/google-readonly/internal/format"
)

func newGetCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "get <file-id>",
		Short: "Get file details",
		Long: `Get detailed metadata for a specific file in Google Drive.

Examples:
  gro drive get <file-id>        # Show file details
  gro drive get <file-id> --json # Output as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("failed to create Drive client: %w", err)
			}

			fileID := args[0]
			file, err := client.GetFile(fileID)
			if err != nil {
				return fmt.Errorf("failed to get file %s: %w", fileID, err)
			}

			if jsonOutput {
				return printJSON(file)
			}

			printFileDetails(file)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

// printFileDetails prints detailed file metadata in a formatted layout
func printFileDetails(f *drive.File) {
	fmt.Println("File Details")
	fmt.Println("────────────────────────────────────────")

	fmt.Printf("ID:         %s\n", f.ID)
	fmt.Printf("Name:       %s\n", f.Name)
	fmt.Printf("Type:       %s\n", drive.GetTypeName(f.MimeType))

	if f.Size > 0 {
		fmt.Printf("Size:       %s\n", format.Size(f.Size))
	} else {
		fmt.Printf("Size:       -\n")
	}

	if !f.CreatedTime.IsZero() {
		fmt.Printf("Created:    %s\n", f.CreatedTime.Format("2006-01-02 15:04:05"))
	}

	if !f.ModifiedTime.IsZero() {
		fmt.Printf("Modified:   %s\n", f.ModifiedTime.Format("2006-01-02 15:04:05"))
	}

	if len(f.Owners) > 0 {
		fmt.Printf("Owner:      %s\n", strings.Join(f.Owners, ", "))
	}

	if f.Shared {
		fmt.Printf("Shared:     Yes\n")
	} else {
		fmt.Printf("Shared:     No\n")
	}

	if f.WebViewLink != "" {
		fmt.Printf("Web Link:   %s\n", f.WebViewLink)
	}

	if len(f.Parents) > 0 {
		fmt.Printf("Parent:     %s\n", strings.Join(f.Parents, ", "))
	}
}

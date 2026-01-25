package drive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/drive"
)

func newDownloadCommand() *cobra.Command {
	var (
		output string
		format string
		stdout bool
	)

	cmd := &cobra.Command{
		Use:   "download <file-id>",
		Short: "Download a file",
		Long: `Download a file from Google Drive or export a Google Workspace file.

Regular files (PDFs, images, etc.) are downloaded directly.
Google Workspace files (Docs, Sheets, Slides) must be exported using --format.

Examples:
  gro drive download <file-id>                  # Download regular file
  gro drive download <file-id> -o ./report.pdf  # Download to specific path
  gro drive download <file-id> --format pdf     # Export Google Doc as PDF
  gro drive download <file-id> --format xlsx    # Export Sheet as Excel
  gro drive download <file-id> --stdout         # Write to stdout

Export formats:
  Documents:     pdf, docx, txt, html, md, rtf, odt
  Spreadsheets:  pdf, xlsx, csv, tsv, ods
  Presentations: pdf, pptx, odp
  Drawings:      pdf, png, svg, jpg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDriveClient()
			if err != nil {
				return fmt.Errorf("failed to create Drive client: %w", err)
			}

			fileID := args[0]

			// Get file metadata first
			file, err := client.GetFile(fileID)
			if err != nil {
				return fmt.Errorf("failed to get file info: %w", err)
			}

			var data []byte

			if drive.IsGoogleWorkspaceFile(file.MimeType) {
				// Google Workspace file - must export
				if format == "" {
					formats := drive.GetSupportedExportFormats(file.MimeType)
					return fmt.Errorf("google %s requires --format flag (supported: %s)",
						drive.GetTypeName(file.MimeType), strings.Join(formats, ", "))
				}

				exportMime, err := drive.GetExportMimeType(file.MimeType, format)
				if err != nil {
					return fmt.Errorf("failed to get export type: %w", err)
				}

				if !stdout {
					fmt.Printf("Exporting: %s\n", file.Name)
					fmt.Printf("Format: %s\n", format)
				}

				data, err = client.ExportFile(fileID, exportMime)
				if err != nil {
					return fmt.Errorf("failed to export file: %w", err)
				}
			} else {
				// Regular file - download directly
				if format != "" {
					return fmt.Errorf("--format flag is only for Google Workspace files; %s is a %s",
						file.Name, drive.GetTypeName(file.MimeType))
				}

				if !stdout {
					fmt.Printf("Downloading: %s\n", file.Name)
				}

				data, err = client.DownloadFile(fileID)
				if err != nil {
					return fmt.Errorf("failed to download file: %w", err)
				}
			}

			// Output to stdout or file
			if stdout {
				_, err = os.Stdout.Write(data)
				if err != nil {
					return fmt.Errorf("failed to write to stdout: %w", err)
				}
				return nil
			}

			outputPath := determineOutputPath(file.Name, format, output)

			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			fmt.Printf("Size: %s\n", formatSize(int64(len(data))))
			fmt.Printf("Saved to: %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&format, "format", "f", "", "Export format for Google Workspace files")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Write to stdout instead of file")

	return cmd
}

// determineOutputPath figures out where to save the downloaded file
func determineOutputPath(originalName, format, userOutput string) string {
	if userOutput != "" {
		return userOutput
	}

	// If exporting with a format, replace the extension
	if format != "" {
		// Remove any existing extension and add the new one
		baseName := strings.TrimSuffix(originalName, filepath.Ext(originalName))
		return baseName + drive.GetFileExtension(format)
	}

	return originalName
}

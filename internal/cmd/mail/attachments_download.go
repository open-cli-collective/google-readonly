package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-readonly/internal/gmail"
	ziputil "github.com/open-cli-collective/google-readonly/internal/zip"
)

var (
	downloadFilename string
	downloadDir      string
	downloadExtract  bool
	downloadAll      bool
)

func newDownloadAttachmentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <message-id>",
		Short: "Download attachments from a message",
		Long: `Download attachments from a Gmail message to local disk.

By default, requires --filename to specify which attachment to download,
or --all to download all attachments.

Zip files can be automatically extracted with --extract flag.

Examples:
  gro mail attachments download 18abc123def456 --filename report.pdf
  gro mail attachments download 18abc123def456 --all
  gro mail attachments download 18abc123def456 --all --output ~/Downloads
  gro mail attachments download 18abc123def456 --filename archive.zip --extract`,
		Args: cobra.ExactArgs(1),
		RunE: runDownloadAttachments,
	}

	cmd.Flags().StringVarP(&downloadFilename, "filename", "f", "",
		"Download only attachment with this filename")
	cmd.Flags().StringVarP(&downloadDir, "output", "o", ".",
		"Directory to save attachments")
	cmd.Flags().BoolVarP(&downloadExtract, "extract", "e", false,
		"Extract zip files after download")
	cmd.Flags().BoolVarP(&downloadAll, "all", "a", false,
		"Download all attachments (required if no --filename specified)")

	return cmd
}

func runDownloadAttachments(cmd *cobra.Command, args []string) error {
	if downloadFilename == "" && !downloadAll {
		return fmt.Errorf("must specify --filename or --all")
	}

	client, err := newGmailClient()
	if err != nil {
		return err
	}

	messageID := args[0]
	attachments, err := client.GetAttachments(messageID)
	if err != nil {
		return err
	}

	if len(attachments) == 0 {
		fmt.Println("No attachments found for message.")
		return nil
	}

	// Filter by filename if specified
	var toDownload []*gmail.Attachment
	for _, att := range attachments {
		if downloadFilename == "" || att.Filename == downloadFilename {
			toDownload = append(toDownload, att)
		}
	}

	if len(toDownload) == 0 {
		return fmt.Errorf("attachment not found: %s", downloadFilename)
	}

	// Create output directory if needed
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get absolute path of download directory for path validation
	absDownloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return fmt.Errorf("failed to resolve download directory: %w", err)
	}

	// Download each attachment
	for _, att := range toDownload {
		// Sanitize filename for display to prevent terminal injection
		safeFilename := SanitizeFilename(att.Filename)

		// Security: Validate output path to prevent path traversal attacks
		outputPath, err := safeOutputPath(absDownloadDir, att.Filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", safeFilename, err)
			continue
		}

		data, err := downloadAttachment(client, messageID, att)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", safeFilename, err)
			continue
		}

		if err := saveAttachment(outputPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", safeFilename, err)
			continue
		}

		fmt.Printf("Downloaded: %s (%s)\n", outputPath, formatSize(int64(len(data))))

		// Extract if zip and --extract flag
		if downloadExtract && isZipFile(att.Filename, att.MimeType) {
			extractDir := filepath.Join(downloadDir,
				strings.TrimSuffix(att.Filename, filepath.Ext(att.Filename)))
			if err := ziputil.Extract(outputPath, extractDir, ziputil.DefaultOptions()); err != nil {
				fmt.Fprintf(os.Stderr, "Error extracting %s: %v\n", safeFilename, err)
			} else {
				fmt.Printf("Extracted to: %s\n", extractDir)
			}
		}
	}

	return nil
}

func downloadAttachment(client gmail.GmailClientInterface, messageID string, att *gmail.Attachment) ([]byte, error) {
	if att.AttachmentID != "" {
		return client.DownloadAttachment(messageID, att.AttachmentID)
	}
	return client.DownloadInlineAttachment(messageID, att.PartID)
}

func saveAttachment(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func isZipFile(filename, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".zip" ||
		mimeType == "application/zip" ||
		mimeType == "application/x-zip-compressed"
}

// safeOutputPath validates that the output path for a filename stays within the
// destination directory, preventing path traversal attacks from malicious filenames.
func safeOutputPath(destDir, filename string) (string, error) {
	// Clean the filename to normalize path separators and remove redundant elements
	cleanName := filepath.Clean(filename)

	// Reject absolute paths
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("invalid attachment filename: absolute path not allowed")
	}

	// Reject paths that try to escape with ..
	if strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return "", fmt.Errorf("invalid attachment filename: path traversal not allowed")
	}

	// Also check for .. anywhere in the path (after cleaning)
	for _, part := range strings.Split(cleanName, string(filepath.Separator)) {
		if part == ".." {
			return "", fmt.Errorf("invalid attachment filename: path traversal not allowed")
		}
	}

	// Build the full output path
	outputPath := filepath.Join(destDir, cleanName)

	// Final security check: ensure the resolved path is within destDir
	// This handles edge cases like symlinks or other path manipulation
	cleanOutput := filepath.Clean(outputPath)
	if !strings.HasPrefix(cleanOutput, destDir+string(filepath.Separator)) && cleanOutput != destDir {
		return "", fmt.Errorf("invalid attachment filename: path escapes destination directory")
	}

	return outputPath, nil
}

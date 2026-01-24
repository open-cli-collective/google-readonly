package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize is the maximum size of a single extracted file (100MB)
	MaxFileSize = 100 * 1024 * 1024
	// MaxTotalSize is the maximum total extracted size (500MB)
	MaxTotalSize = 500 * 1024 * 1024
	// MaxFiles is the maximum number of files to extract
	MaxFiles = 1000
	// MaxDepth is the maximum nesting depth for extracted directories
	MaxDepth = 10
)

// Options configures zip extraction behavior
type Options struct {
	MaxFileSize  int64
	MaxTotalSize int64
	MaxFiles     int
	MaxDepth     int
}

// DefaultOptions returns safe default extraction options
func DefaultOptions() Options {
	return Options{
		MaxFileSize:  MaxFileSize,
		MaxTotalSize: MaxTotalSize,
		MaxFiles:     MaxFiles,
		MaxDepth:     MaxDepth,
	}
}

// Extract safely extracts a zip file to the destination directory
func Extract(zipPath, destDir string, opts Options) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Validate before extraction
	if err := validateZip(&r.Reader, opts); err != nil {
		return err
	}

	// Create destination directory
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}
	if err := fs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	var totalSize int64
	for _, f := range r.File {
		if err := extractFile(f, destDir, opts, &totalSize); err != nil {
			return err
		}
	}

	return nil
}

func validateZip(r *zip.Reader, opts Options) error {
	if len(r.File) > opts.MaxFiles {
		return fmt.Errorf("zip contains too many files: %d (max %d)",
			len(r.File), opts.MaxFiles)
	}

	var totalSize uint64
	for _, f := range r.File {
		// Check for zip bomb (compression ratio attack)
		if f.UncompressedSize64 > uint64(opts.MaxFileSize) {
			return fmt.Errorf("file %s exceeds max size: %d bytes",
				f.Name, f.UncompressedSize64)
		}
		totalSize += f.UncompressedSize64
	}

	if totalSize > uint64(opts.MaxTotalSize) {
		return fmt.Errorf("total extracted size exceeds limit: %d bytes (max %d)",
			totalSize, opts.MaxTotalSize)
	}

	return nil
}

func extractFile(f *zip.File, destDir string, opts Options, totalSize *int64) error {
	// Security: Prevent path traversal attacks
	name := filepath.Clean(f.Name)

	// Reject absolute paths and paths starting with ..
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
		return fmt.Errorf("invalid file path in zip: %s", f.Name)
	}

	// Check nesting depth
	depth := strings.Count(name, string(os.PathSeparator))
	if depth > opts.MaxDepth {
		return fmt.Errorf("file path too deep: %s (depth %d, max %d)",
			name, depth, opts.MaxDepth)
	}

	destPath := filepath.Join(destDir, name)

	// Security: Ensure the destination is within destDir (handles symlink attacks)
	cleanDest := filepath.Clean(destPath)
	if !strings.HasPrefix(cleanDest, destDir+string(os.PathSeparator)) && cleanDest != destDir {
		return fmt.Errorf("path traversal detected: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return fs.MkdirAll(destPath, f.Mode())
	}

	// Create parent directories
	if err := fs.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Extract file with size limit
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := fs.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Use LimitedReader to enforce size limits during extraction
	limitedReader := &io.LimitedReader{R: rc, N: opts.MaxFileSize + 1}
	written, err := io.Copy(outFile, limitedReader)
	if err != nil {
		return err
	}

	if written > opts.MaxFileSize {
		_ = fs.Remove(destPath)
		return fmt.Errorf("file %s exceeds max size during extraction", f.Name)
	}

	*totalSize += written
	if *totalSize > opts.MaxTotalSize {
		return fmt.Errorf("total extracted size exceeds limit")
	}

	return nil
}

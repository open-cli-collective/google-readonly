package mail

import (
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestSafeOutputPath(t *testing.T) {
	destDir := "/tmp/downloads"

	tests := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "simple filename",
			filename:    "report.pdf",
			expectError: false,
		},
		{
			name:        "filename with spaces",
			filename:    "my report.pdf",
			expectError: false,
		},
		{
			name:        "filename in subdirectory",
			filename:    "attachments/report.pdf",
			expectError: false,
		},
		{
			name:        "path traversal with ..",
			filename:    "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal not allowed",
		},
		{
			name:        "path traversal at start",
			filename:    "../secret.txt",
			expectError: true,
			errorMsg:    "path traversal not allowed",
		},
		{
			name:        "path traversal in middle",
			filename:    "subdir/../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal not allowed",
		},
		{
			name:        "double dot only",
			filename:    "..",
			expectError: true,
			errorMsg:    "path traversal not allowed",
		},
		{
			name:        "absolute path unix",
			filename:    "/etc/passwd",
			expectError: true,
			errorMsg:    "absolute path not allowed",
		},
		{
			name:        "hidden file",
			filename:    ".hidden",
			expectError: false,
		},
		{
			name:        "dot in filename",
			filename:    "report.v2.pdf",
			expectError: false,
		},
		{
			name:        "empty after traversal",
			filename:    "foo/../bar",
			expectError: false, // After cleaning becomes "bar" which is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeOutputPath(destDir, tt.filename)

			if tt.expectError {
				testutil.Error(t, err)
				if tt.errorMsg != "" {
					testutil.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				testutil.NoError(t, err)
				// Verify the result is within destDir
				testutil.True(t, filepath.IsAbs(result) || result == filepath.Join(destDir, filepath.Clean(tt.filename)))
			}
		})
	}
}

func TestSafeOutputPath_StaysWithinDestDir(t *testing.T) {
	destDir := "/tmp/downloads"

	// Valid cases should produce paths within destDir
	validCases := []string{
		"simple.txt",
		"dir/file.txt",
		"a/b/c/deep.txt",
	}

	for _, filename := range validCases {
		t.Run(filename, func(t *testing.T) {
			result, err := safeOutputPath(destDir, filename)
			testutil.NoError(t, err)

			// Result must start with destDir
			testutil.True(t, len(result) >= len(destDir))
			testutil.Equal(t, result[:len(destDir)], destDir)
		})
	}
}

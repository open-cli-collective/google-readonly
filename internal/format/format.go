// Package format provides shared formatting utilities for consistent output.
package format

import "fmt"

// Truncate shortens a string to maxLen characters, adding "..." if truncated.
// If the string is already within maxLen, it is returned unchanged.
func Truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4 // Minimum length to fit "..."
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Size converts bytes to human-readable format (e.g., "1.5 KB", "2.3 MB").
func Size(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

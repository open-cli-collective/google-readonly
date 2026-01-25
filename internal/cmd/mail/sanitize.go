package mail

import (
	"regexp"
	"strings"
)

// ansiEscapeRegex matches ANSI escape sequences including:
// - CSI sequences (ESC [ ... letter) - cursor movement, colors, etc.
// - OSC sequences (ESC ] ... BEL/ST) - window title, hyperlinks, etc.
// - Simple escape sequences (ESC followed by single character)
var ansiEscapeRegex = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*\x07|\][^\x1b]*\x1b\\|[@-Z\\-_])`)

// controlCharRegex matches other potentially dangerous control characters
// excluding common whitespace (tab, newline, carriage return)
var controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1a\x1c-\x1f\x7f]`)

// SanitizeOutput removes ANSI escape sequences and dangerous control characters
// from a string to prevent terminal injection attacks. Safe whitespace characters
// (tab, newline, carriage return) are preserved.
func SanitizeOutput(s string) string {
	// Remove ANSI escape sequences
	s = ansiEscapeRegex.ReplaceAllString(s, "")

	// Remove other control characters (except tab, newline, carriage return)
	s = controlCharRegex.ReplaceAllString(s, "")

	return s
}

// SanitizeFilename sanitizes a filename for display, removing potentially
// dangerous characters while preserving readability.
func SanitizeFilename(s string) string {
	// First apply general output sanitization
	s = SanitizeOutput(s)

	// Additionally handle Unicode direction overrides that could be used
	// to disguise file extensions (e.g., making "evil.exe" appear as "exe.live")
	// Unicode bidirectional control characters
	bidiChars := []string{
		"\u202A", // LEFT-TO-RIGHT EMBEDDING
		"\u202B", // RIGHT-TO-LEFT EMBEDDING
		"\u202C", // POP DIRECTIONAL FORMATTING
		"\u202D", // LEFT-TO-RIGHT OVERRIDE
		"\u202E", // RIGHT-TO-LEFT OVERRIDE
		"\u2066", // LEFT-TO-RIGHT ISOLATE
		"\u2067", // RIGHT-TO-LEFT ISOLATE
		"\u2068", // FIRST STRONG ISOLATE
		"\u2069", // POP DIRECTIONAL ISOLATE
	}

	for _, char := range bidiChars {
		s = strings.ReplaceAll(s, char, "")
	}

	return s
}

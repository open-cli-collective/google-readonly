// Package log provides simple structured logging with verbosity levels.
// Debug messages are only shown when Verbose is true.
// All other messages always print to stderr.
package log

import (
	"fmt"
	"os"
)

// Verbose controls whether Debug messages are printed.
// Set this via the root command's --verbose flag.
var Verbose bool

// Debug prints a debug message to stderr if Verbose is true.
// Format follows fmt.Printf conventions.
func Debug(format string, args ...any) {
	if !Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}

// Info prints an informational message to stderr.
// Format follows fmt.Printf conventions.
func Info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Warn prints a warning message to stderr.
// Format follows fmt.Printf conventions.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// Error prints an error message to stderr.
// Format follows fmt.Printf conventions.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

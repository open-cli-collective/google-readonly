// Package view provides a small, consistent output helper for interactive CLI
// flows (Success / Error / Info / Printf / Println). Color is via lipgloss and
// honors NO_COLOR.
package view

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// View writes status messages to stdout/stderr with optional color.
type View struct {
	Out io.Writer
	Err io.Writer

	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	infoStyle    lipgloss.Style
}

// New returns a View writing to os.Stdout / os.Stderr.
func New() *View {
	return NewWithWriters(os.Stdout, os.Stderr)
}

// NewWithWriters returns a View writing to the provided writers.
func NewWithWriters(out, err io.Writer) *View {
	return &View{
		Out:          out,
		Err:          err,
		successStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		infoStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}

// Success writes a green-prefixed line to stdout.
func (v *View) Success(format string, args ...any) {
	prefix := v.successStyle.Render("✓")
	_, _ = fmt.Fprintf(v.Out, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Error writes a red-prefixed line to stderr.
func (v *View) Error(format string, args ...any) {
	prefix := v.errorStyle.Render("✗")
	_, _ = fmt.Fprintf(v.Err, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Info writes a dim-prefixed line to stdout.
func (v *View) Info(format string, args ...any) {
	prefix := v.infoStyle.Render("·")
	_, _ = fmt.Fprintf(v.Out, "%s %s\n", prefix, fmt.Sprintf(format, args...))
}

// Printf writes a formatted line to stdout (no prefix, no automatic newline).
func (v *View) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(v.Out, format, args...)
}

// Println writes the string to stdout followed by a newline.
func (v *View) Println(s string) {
	_, _ = fmt.Fprintln(v.Out, s)
}

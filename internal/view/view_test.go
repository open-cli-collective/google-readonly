package view

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestSuccessGoesToStdout(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Success("did the thing %d", 42)
	if !strings.Contains(out.String(), "did the thing 42") {
		t.Fatalf("expected message in stdout, got %q", out.String())
	}
	if errw.Len() != 0 {
		t.Fatalf("expected stderr empty, got %q", errw.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("expected trailing newline, got %q", out.String())
	}
}

func TestErrorGoesToStderr(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Error("nope: %s", "boom")
	if !strings.Contains(errw.String(), "nope: boom") {
		t.Fatalf("expected message in stderr, got %q", errw.String())
	}
	if out.Len() != 0 {
		t.Fatalf("expected stdout empty, got %q", out.String())
	}
}

func TestInfoGoesToStdout(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Info("hint: %s", "something")
	if !strings.Contains(out.String(), "hint: something") {
		t.Fatalf("expected message in stdout, got %q", out.String())
	}
}

func TestPrintfNoTrailingNewline(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Printf("a=%d", 1)
	if got := out.String(); got != "a=1" {
		t.Fatalf("expected exact 'a=1', got %q", got)
	}
}

func TestPrintlnAddsNewline(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Println("hello")
	if got := out.String(); got != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", got)
	}
}

// withRenderer swaps the lipgloss default renderer for the duration of the
// test, restoring the saved renderer on cleanup. Tests using this must not
// call t.Parallel — the default renderer is process-global.
func withRenderer(t *testing.T, profile termenv.Profile) {
	t.Helper()
	saved := lipgloss.DefaultRenderer()
	t.Cleanup(func() { lipgloss.SetDefaultRenderer(saved) })

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	lipgloss.SetDefaultRenderer(r)
}

func TestSuccessUnderAsciiProfileEmitsNoANSI(t *testing.T) {
	withRenderer(t, termenv.Ascii)

	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Success("hello")

	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("expected no ANSI escape under ascii profile, got %q", out.String())
	}
}

func TestSuccessUnderANSIProfileEmitsANSI(t *testing.T) {
	// Regression: proves the no-color test isn't just env-determined.
	withRenderer(t, termenv.ANSI)

	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Success("hello")

	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("expected ANSI escape under ANSI profile, got %q", out.String())
	}
}

func TestErrorUnderAsciiProfileEmitsNoANSI(t *testing.T) {
	withRenderer(t, termenv.Ascii)

	var out, errw bytes.Buffer
	v := NewWithWriters(&out, &errw)
	v.Error("bad")

	if strings.Contains(errw.String(), "\x1b[") {
		t.Fatalf("expected no ANSI escape under ascii profile, got %q", errw.String())
	}
}

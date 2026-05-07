package view

import (
	"bytes"
	"strings"
	"testing"
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

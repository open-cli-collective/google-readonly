package log

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDebug_WhenVerboseTrue(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Enable verbose
	oldVerbose := Verbose
	Verbose = true
	defer func() { Verbose = oldVerbose }()

	Debug("test message %d", 42)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("expected %q to contain %q", output, "[DEBUG]")
	}
	if !strings.Contains(output, "test message 42") {
		t.Errorf("expected %q to contain %q", output, "test message 42")
	}
}

func TestDebug_WhenVerboseFalse(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Disable verbose
	oldVerbose := Verbose
	Verbose = false
	defer func() { Verbose = oldVerbose }()

	Debug("should not appear")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("got %q, want empty string", output)
	}
}

func TestInfo(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Info("info message %s", "test")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "info message test\n" {
		t.Errorf("got %v, want %v", output, "info message test\n")
	}
	if strings.Contains(output, "[INFO]") {
		t.Error("got true, want false")
	} // No prefix for info
}

func TestWarn(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Warn("warning: %s", "something")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[WARN]") {
		t.Errorf("expected %q to contain %q", output, "[WARN]")
	}
	if !strings.Contains(output, "warning: something") {
		t.Errorf("expected %q to contain %q", output, "warning: something")
	}
}

func TestError(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Error("error occurred: %v", "failure")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("expected %q to contain %q", output, "[ERROR]")
	}
	if !strings.Contains(output, "error occurred: failure") {
		t.Errorf("expected %q to contain %q", output, "error occurred: failure")
	}
}

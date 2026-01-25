package log

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Contains(t, output, "[DEBUG]")
	assert.Contains(t, output, "test message 42")
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

	assert.Empty(t, output)
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

	assert.Equal(t, "info message test\n", output)
	assert.False(t, strings.Contains(output, "[INFO]")) // No prefix for info
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

	assert.Contains(t, output, "[WARN]")
	assert.Contains(t, output, "warning: something")
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

	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "error occurred: failure")
}

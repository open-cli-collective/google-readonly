package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureStdout captures everything written to os.Stdout during the execution
// of f and returns it as a string. This is useful for testing commands that
// print output directly to stdout.
func CaptureStdout(t testing.TB, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	NoError(t, err)
	os.Stdout = w

	f()

	// Close error is non-fatal for pipe operations in tests
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// WithFactory temporarily replaces a factory function variable with a
// replacement value, executes f, then restores the original. This is the
// generic building block for per-package withMockClient helpers.
//
// Usage:
//
//	testutil.WithFactory(&ClientFactory, mockFactory, func() {
//	    // ClientFactory now returns the mock
//	})
func WithFactory[T any](factoryPtr *T, replacement T, f func()) {
	original := *factoryPtr
	*factoryPtr = replacement
	defer func() { *factoryPtr = original }()
	f()
}

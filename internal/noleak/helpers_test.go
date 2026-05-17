package noleak

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStdout redirects the process os.Stdout (the gro config commands
// print via fmt.Printf, not cmd.OutOrStdout) for the duration of f and
// returns everything written. Serialized by the Go test runner per package;
// these tests do not run in parallel.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		f()
	}()
	return <-done
}

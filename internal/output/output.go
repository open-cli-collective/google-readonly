// Package output provides shared output utilities for formatting and printing
// data to various destinations.
package output

import (
	"encoding/json"
	"io"
	"os"
)

// JSON encodes data as indented JSON to the given writer.
func JSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// JSONStdout encodes data as indented JSON to stdout.
// This is a convenience function that wraps JSON(os.Stdout, data).
func JSONStdout(data any) error {
	return JSON(os.Stdout, data)
}

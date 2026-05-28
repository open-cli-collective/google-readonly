// Package output provides shared output utilities for formatting and printing
// data to various destinations.
package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/open-cli-collective/google-readonly/internal/migrationsink"
)

// JSON encodes data as indented JSON to the given writer. If a one-time
// §1.8 migration block is pending, it is spliced in as a top-level
// "_migration" field and consumed (so it appears exactly once, on the first
// JSON command after the migration). With no pending block the output is
// byte-identical to a plain indented Encoder (2-space indent, trailing
// newline) — the surviving control-plane JSONStdout callers (refresh,
// config show) inherit the splice for free.
func JSON(w io.Writer, data any) error {
	mig, _ := migrationsink.Take()
	if mig == nil {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, spliceMigration(body, mig), "", "  "); err != nil {
		return err
	}
	pretty.WriteByte('\n')
	_, err = w.Write(pretty.Bytes())
	return err
}

// JSONStdout encodes data as indented JSON to stdout.
// This is a convenience function that wraps JSON(os.Stdout, data).
func JSONStdout(data any) error {
	return JSON(os.Stdout, data)
}

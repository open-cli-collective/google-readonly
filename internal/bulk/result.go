package bulk

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/google-readonly/internal/output"
)

// Result represents the outcome of a bulk operation.
type Result struct {
	Action  string   `json:"action"`
	IDs     []string `json:"ids"`
	Count   int      `json:"count"`
	DryRun  bool     `json:"dryRun"`
	Details any      `json:"details,omitempty"`
}

// Print outputs the result as text or JSON.
func (r *Result) Print(jsonOutput bool) error {
	if jsonOutput {
		return output.JSONStdout(r)
	}
	if r.DryRun {
		fmt.Printf("[dry-run] Would %s %d message(s).\n", r.Action, r.Count)
	} else {
		fmt.Printf("%s %d message(s).\n", capitalize(r.Action), r.Count)
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

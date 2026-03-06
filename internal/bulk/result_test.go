package bulk

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestResult_Print_Text(t *testing.T) {
	r := &Result{Action: "archived", IDs: []string{"a", "b"}, Count: 2}
	out := testutil.CaptureStdout(t, func() {
		err := r.Print(false)
		testutil.NoError(t, err)
	})
	testutil.Contains(t, out, "Archived 2 message(s).")
}

func TestResult_Print_TextDryRun(t *testing.T) {
	r := &Result{Action: "archive", IDs: []string{"a", "b"}, Count: 2, DryRun: true}
	out := testutil.CaptureStdout(t, func() {
		err := r.Print(false)
		testutil.NoError(t, err)
	})
	testutil.Contains(t, out, "[dry-run] Would archive 2 message(s).")
}

func TestResult_Print_JSON(t *testing.T) {
	r := &Result{Action: "archived", IDs: []string{"a"}, Count: 1}
	out := testutil.CaptureStdout(t, func() {
		err := r.Print(true)
		testutil.NoError(t, err)
	})
	var parsed Result
	err := json.Unmarshal([]byte(out), &parsed)
	testutil.NoError(t, err)
	testutil.Equal(t, parsed.Action, "archived")
	testutil.Equal(t, parsed.Count, 1)
}

func TestCapitalize(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, capitalize("archived"), "Archived")
	testutil.Equal(t, capitalize("starred"), "Starred")
	testutil.Equal(t, capitalize(""), "")
}

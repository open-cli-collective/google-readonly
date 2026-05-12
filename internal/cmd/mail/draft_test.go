package mail

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// --- Flag presence ---

func TestDraftCommand_FlagsPresent(t *testing.T) {
	cmd := newDraftCommand()
	for _, name := range []string{
		"to", "cc", "bcc", "from", "subject",
		"body", "stdin", "file", "plain", "html",
		"attach", "json",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q missing", name)
		}
	}
}

// --- Validation negative cases ---

func TestDraftCommand_RequiresTo(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--subject", "hi", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--to")
	})
}

func TestDraftCommand_RequiresSubject(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--subject")
	})
}

func TestDraftCommand_EmptySubjectAllowedIfExplicit(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "", "--body", "x"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, seen.Subject, "")
	})
}

func TestDraftCommand_RequiresOneBodySource(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "exactly one of")
	})
}

func TestDraftCommand_RejectsMultipleBodySources(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi", "--body", "x", "--file", "/dev/null"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "exactly one of")
	})
}

func TestDraftCommand_RejectsPlainAndHTMLTogether(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi", "--body", "x", "--plain", "--html"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		// Pin the full message so the test catches a regression where only
		// "--plain" appears (e.g. if the error wording for an unrelated flag
		// happened to mention --plain).
		testutil.Contains(t, err.Error(), "--plain")
		testutil.Contains(t, err.Error(), "--html")
		testutil.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestDraftCommand_RejectsInvalidFrom(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi", "--body", "x", "--from", "not-an-email"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--from")
	})
}

func TestDraftCommand_RejectsInvalidTo(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "not-an-email", "--subject", "hi", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--to")
	})
}

func TestDraftCommand_RejectsHeaderInjectionInTo(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com\r\nBcc: evil@x.com", "--subject", "hi", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "CR or LF")
	})
}

func TestDraftCommand_RejectsHeaderInjectionInSubject(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi\r\nBcc: x@y.com", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "CR or LF")
	})
}

func TestDraftCommand_RejectsEmptyTo(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "", "--subject", "hi", "--body", "x"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		// Distinct code path from "missing --to": flag is Changed but parses
		// to zero addresses, so the len() == 0 guard fires.
		testutil.Contains(t, err.Error(), "--to")
	})
}

func TestDraftCommand_RejectsMissingFileBody(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "hi", "--file", "/nope/missing-body.md"})
	withMockClient(&MockGmailClient{
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called when --file is unreadable")
			return nil, nil
		},
	}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "reading body file")
	})
}

// --- Success paths ---

func TestDraftCommand_DefaultIsMarkdownRenderedToHTML(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1", MessageID: "m1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "alice@example.com", "--subject", "Hi", "--body", "**bold** and _italic_"})
	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyHTML)
		testutil.Contains(t, string(seen.Body), "<strong>bold</strong>")
		testutil.Contains(t, string(seen.Body), "<em>italic</em>")
		testutil.Contains(t, output, "Draft created: d1")
	})
}

func TestDraftCommand_PlainSendsPlainText(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "**bold**", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyPlainText)
		testutil.Equal(t, string(seen.Body), "**bold**") // verbatim, no rendering
	})
}

func TestDraftCommand_HTMLPassesThroughVerbatim(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "<h1>Raw</h1>", "--html"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyHTML)
		testutil.Equal(t, string(seen.Body), "<h1>Raw</h1>") // verbatim
	})
}

func TestDraftCommand_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.md")
	if err := os.WriteFile(path, []byte("# hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--file", path})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Contains(t, string(seen.Body), "<h1>hello</h1>")
	})
}

func TestDraftCommand_Stdin(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetIn(strings.NewReader("## from stdin"))
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--stdin"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Contains(t, string(seen.Body), "<h2>from stdin</h2>")
	})
}

func TestDraftCommand_Attachments_BasenameOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret-report.pdf")
	if err := os.WriteFile(path, []byte("PDFBYTES"), 0o600); err != nil {
		t.Fatal(err)
	}

	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "See attached", "--body", "here", "--attach", path})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		if len(seen.Attachments) != 1 {
			t.Fatalf("attachments = %d, want 1", len(seen.Attachments))
		}
		testutil.Equal(t, seen.Attachments[0].Filename, "secret-report.pdf") // basename only
		testutil.Equal(t, string(seen.Attachments[0].Data), "PDFBYTES")
		// mime.TypeByExtension can return either "application/pdf" or
		// "application/pdf; charset=binary" depending on OS MIME database.
		// Prefix-match to stay portable across macOS/Linux CI runners.
		got := seen.Attachments[0].MimeType
		if got != "application/pdf" && !strings.HasPrefix(got, "application/pdf;") {
			t.Errorf("MimeType = %q, want application/pdf (with optional params)", got)
		}
	})
}

func TestDraftCommand_MultipleRecipients(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{
		"--to", "a@x.com, b@x.com",
		"--cc", "c@x.com",
		"--bcc", "d@x.com, e@x.com",
		"--subject", "Hi", "--body", "text",
	})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, len(seen.To), 2)
		testutil.Equal(t, len(seen.Cc), 1)
		testutil.Equal(t, len(seen.Bcc), 2)
	})
}

func TestDraftCommand_AttachmentMissingFile(t *testing.T) {
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft should not be called when attachment file is missing")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "x", "--attach", "/nope/does/not/exist.pdf"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		// Pin both the failing operation and the path so this confirms
		// attachment loading is the actual failure point.
		testutil.Contains(t, err.Error(), "reading attachment")
		testutil.Contains(t, err.Error(), "/nope/does/not/exist.pdf")
	})
}

func TestDraftCommand_JSONOutput(t *testing.T) {
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			return &gmailapi.DraftResult{ID: "d1", MessageID: "m1", ThreadID: "t1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "x", "--json"})
	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			testutil.NoError(t, err)
		})
		var result gmailapi.DraftResult
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("unmarshal: %v\noutput=%q", err, output)
		}
		testutil.Equal(t, result.ID, "d1")
		testutil.Equal(t, result.MessageID, "m1")
		testutil.Equal(t, result.ThreadID, "t1")
	})
}

func TestDraftCommand_APIError(t *testing.T) {
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			return nil, &mockErr{msg: "boom"}
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "x"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "creating draft")
	})
}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

// --- Codex review: cobra.NoArgs ---

func TestDraftCommand_RejectsPositionalArgs(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"accidental", "--to", "a@x.com", "--subject", "hi", "--body", "x"})
	withMockClient(&MockGmailClient{
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft should not be called when positional args are present")
			return nil, nil
		},
	}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
	})
}

// --- Codex review: --from normalisation ---

func TestDraftCommand_FromNormalisesDisplayName(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--from", "Work <work@me.com>", "--to", "a@x.com", "--subject", "Hi", "--body", "x"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		// Codex M2: raw value "Work <work@me.com>" must be normalised to
		// "work@me.com" before reaching DraftMessage.
		testutil.Equal(t, seen.From, "work@me.com")
	})
}

// --- TDD review: command-layer CR/LF guard across all address fields ---

func TestDraftCommand_RejectsHeaderInjection_AllFields(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"--cc CRLF", []string{"--to", "a@x.com", "--cc", "b@x.com\r\nBcc: e@x.com", "--subject", "hi", "--body", "x"}},
		{"--bcc LF", []string{"--to", "a@x.com", "--bcc", "b@x.com\nBcc: e@x.com", "--subject", "hi", "--body", "x"}},
		{"--from CRLF", []string{"--to", "a@x.com", "--from", "a@x.com\r\nBcc: e@x.com", "--subject", "hi", "--body", "x"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := newDraftCommand()
			cmd.SetArgs(tc.args)
			withMockClient(&MockGmailClient{}, func() {
				err := cmd.Execute()
				testutil.Error(t, err)
			})
		})
	}
}

// --- TDD review: detectMimeType fallback branches ---

func TestDetectMimeType(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		want     string // empty = "anything but the default", expects known type
		fallback bool
	}{
		{"known extension", "report.pdf", "application/pdf", false},
		{"no extension", "Makefile", "application/octet-stream", true},
		{"unknown extension", "data.xyzunknown", "application/octet-stream", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := detectMimeType(tc.path)
			if tc.fallback {
				testutil.Equal(t, got, "application/octet-stream")
				return
			}
			// Known type: prefix-match so "application/pdf" or
			// "application/pdf; charset=..." both pass.
			if got != tc.want && !strings.HasPrefix(got, tc.want+";") {
				t.Errorf("detectMimeType(%q) = %q, want %q (or with params)", tc.path, got, tc.want)
			}
		})
	}
}

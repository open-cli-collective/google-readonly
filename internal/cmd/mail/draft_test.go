package mail

import (
	"context"
	"encoding/json"
	"fmt"
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
		"--subject", "Hi", "--body", "text", "--plain",
	})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, len(seen.To), 2)
		testutil.Equal(t, len(seen.Cc), 1)
		testutil.Equal(t, len(seen.Bcc), 2)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyPlainText)
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
				testutil.Contains(t, err.Error(), "CR or LF")
			})
		})
	}
}

func TestDetectMimeType_UppercaseExtension(t *testing.T) {
	if got := detectMimeType("Report.PDF"); got != "application/pdf" && !strings.HasPrefix(got, "application/pdf;") {
		t.Errorf("detectMimeType(Report.PDF) = %q, want application/pdf", got)
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

// --- Reply mode tests (#114) ---

// srcReply is a fully-populated synthetic source message for reply tests.
func srcReply() *gmailapi.Message {
	return &gmailapi.Message{
		ID:           "msg-src",
		ThreadID:     "thread-src",
		From:         "Alice <alice@example.com>",
		To:           "me@example.com, bob@example.com",
		Cc:           "carol@example.com",
		Subject:      "Project sync",
		RFCMessageID: "<orig@example.com>",
		References:   "<earlier@example.com>",
	}
}

func TestDraftCommand_ReplyTo_DerivesAllDefaults(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			return srcReply(), nil
		},
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "thanks", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, seen.ThreadID, "thread-src")
		testutil.Equal(t, seen.InReplyTo, "<orig@example.com>")
		testutil.LenSlice(t, len(seen.References), 2)
		testutil.Equal(t, seen.References[0], "<earlier@example.com>")
		testutil.Equal(t, seen.References[1], "<orig@example.com>")
		testutil.LenSlice(t, len(seen.To), 1)
		testutil.Equal(t, seen.To[0], "alice@example.com")
		testutil.LenSlice(t, len(seen.Cc), 0)
		testutil.Equal(t, seen.Subject, "Re: Project sync")
	})
}

func TestDraftCommand_ReplyTo_ExplicitOverrides(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReply(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{
		"--reply-to", "msg-src",
		"--to", "override@x.com",
		"--cc", "extra@x.com",
		"--subject", "Custom subject",
		"--body", "x", "--plain",
	})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.LenSlice(t, len(seen.To), 1)
		testutil.Equal(t, seen.To[0], "override@x.com")
		testutil.LenSlice(t, len(seen.Cc), 1)
		testutil.Equal(t, seen.Cc[0], "extra@x.com")
		testutil.Equal(t, seen.Subject, "Custom subject")
		// Headers and ThreadID still derived even when subject/to overridden.
		testutil.Equal(t, seen.ThreadID, "thread-src")
		testutil.Equal(t, seen.InReplyTo, "<orig@example.com>")
	})
}

func TestDraftCommand_ReplyAll_AddsCc(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReply(), nil },
		GetProfileFunc: func(_ context.Context) (*gmailapi.Profile, error) {
			return &gmailapi.Profile{EmailAddress: "me@example.com"}, nil
		},
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		// Source To has me@example.com (filtered) + bob@example.com; Cc has carol@example.com.
		testutil.LenSlice(t, len(seen.Cc), 2)
		testutil.Equal(t, seen.Cc[0], "bob@example.com")
		testutil.Equal(t, seen.Cc[1], "carol@example.com")
	})
}

func TestDraftCommand_ReplyAll_FiltersFromAlias(t *testing.T) {
	var seen gmailapi.DraftMessage
	src := srcReply()
	src.To = "alias@example.com, bob@example.com"
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		GetProfileFunc: func(_ context.Context) (*gmailapi.Profile, error) {
			return &gmailapi.Profile{EmailAddress: "me@example.com"}, nil
		},
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--from", "alias@example.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		// Source To = alias + bob; source Cc = carol. alias filtered via --from.
		// Pin the exact list so the test fails if filtering accidentally drops bob/carol too.
		testutil.LenSlice(t, len(seen.Cc), 2)
		testutil.Equal(t, seen.Cc[0], "bob@example.com")
		testutil.Equal(t, seen.Cc[1], "carol@example.com")
	})
}

func TestDraftCommand_ReplyTo_OverrideToWithMalformedSourceFrom(t *testing.T) {
	// --to override means the source From is never parsed, so a malformed
	// source From header must not fail the command.
	var seen gmailapi.DraftMessage
	src := srcReply()
	src.From = "not-an-email"
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--to", "override@x.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.LenSlice(t, len(seen.To), 1)
		testutil.Equal(t, seen.To[0], "override@x.com")
		// Threading headers still derive (Message-Id is unaffected).
		testutil.Equal(t, seen.InReplyTo, "<orig@example.com>")
	})
}

func TestDraftCommand_ReplyAll_OverrideCcWithMalformedSourceToCc(t *testing.T) {
	// --cc override means source To/Cc are never parsed, so malformed values must not fail.
	var seen gmailapi.DraftMessage
	src := srcReply()
	src.To = "garbage @@@ not parseable"
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--cc", "explicit@x.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.LenSlice(t, len(seen.Cc), 1)
		testutil.Equal(t, seen.Cc[0], "explicit@x.com")
	})
}

func TestDraftCommand_EmptyReplyToIsAnError(t *testing.T) {
	// --reply-to with an empty value (often an unexpanded shell variable)
	// must fail explicitly rather than silently producing an unthreaded draft.
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			t.Fatal("GetMessage must not be called with empty --reply-to")
			return nil, nil
		},
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called with empty --reply-to")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "", "--to", "a@x.com", "--subject", "hi", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--reply-to requires a non-empty message ID")
	})
}

func TestDraftCommand_ReplyTo_EmptyToOverrideRejectedLocally(t *testing.T) {
	// In reply mode, an explicit --to "" override must fail before any
	// GetMessage call — it's a local user error.
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			t.Fatal("GetMessage must not be called when --to override is empty")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--to", "", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--to must contain at least one address")
	})
}

func TestDraftCommand_ReplyAll_ExplicitToOverride(t *testing.T) {
	// --reply-all with explicit --to: To is user's value (needs.To=false), Cc is derived from source.
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc:  func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReply(), nil },
		GetProfileFunc:  func(_ context.Context) (*gmailapi.Profile, error) { return &gmailapi.Profile{EmailAddress: "me@example.com"}, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) { seen = msg; return &gmailapi.DraftResult{ID: "d1"}, nil },
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--to", "different@x.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.LenSlice(t, len(seen.To), 1)
		testutil.Equal(t, seen.To[0], "different@x.com")
		// Cc still derived from source To+Cc minus me@example.com.
		testutil.LenSlice(t, len(seen.Cc), 2)
		testutil.Equal(t, seen.Cc[0], "bob@example.com")
		testutil.Equal(t, seen.Cc[1], "carol@example.com")
	})
}

func TestBuildReferences_DedupesTrailingMessageID(t *testing.T) {
	// If the source's References header already ends with its own Message-Id
	// (some MUAs do this), the chain shouldn't duplicate.
	got := buildReferences("<a@x> <orig@x>", "<orig@x>")
	if len(got) != 2 || got[0] != "<a@x>" || got[1] != "<orig@x>" {
		t.Errorf("buildReferences dedup = %v, want [<a@x> <orig@x>]", got)
	}
}

func TestDraftCommand_ReplyAll_RequiresReplyTo(t *testing.T) {
	mock := &MockGmailClient{}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "x", "--reply-all"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--reply-all")
		testutil.Contains(t, err.Error(), "--reply-to")
	})
}

func TestDraftCommand_Reply_NoDoublePrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Re: foo", "Re: foo"},
		{"re: foo", "re: foo"},
		{"RE: foo", "RE: foo"},
		{"Aw: foo", "Re: Aw: foo"},
		{"foo", "Re: foo"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			src := srcReply()
			src.Subject = tc.in
			var seen gmailapi.DraftMessage
			mock := &MockGmailClient{
				GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
				CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
					seen = msg
					return &gmailapi.DraftResult{ID: "d1"}, nil
				},
			}
			cmd := newDraftCommand()
			cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
			withMockClient(mock, func() {
				err := cmd.Execute()
				testutil.NoError(t, err)
				testutil.Equal(t, seen.Subject, tc.want)
			})
		})
	}
}

func TestDraftCommand_ReplyTo_GetMessageFails(t *testing.T) {
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			return nil, fmt.Errorf("not found")
		},
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called when GetMessage fails")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "fetching reply source")
	})
}

func TestDraftCommand_ReplyTo_MissingFromHeader(t *testing.T) {
	src := srcReply()
	src.From = ""
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "no From header")
	})
}

func TestDraftCommand_ReplyTo_MissingMessageID(t *testing.T) {
	src := srcReply()
	src.RFCMessageID = ""
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "no Message-Id header")
	})
}

func TestDraftCommand_ReplyTo_MalformedFromHeader(t *testing.T) {
	src := srcReply()
	src.From = "not-an-email"
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "parsing source From header")
	})
}

func TestDraftCommand_ReplyAll_MalformedToHeader(t *testing.T) {
	src := srcReply()
	src.To = "not a valid header @@@"
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		GetProfileFunc: func(_ context.Context) (*gmailapi.Profile, error) {
			return &gmailapi.Profile{EmailAddress: "me@example.com"}, nil
		},
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "parsing source To/Cc headers")
	})
}

func TestDraftCommand_LocalValidationBeforeFetch(t *testing.T) {
	// --body + --stdin is invalid locally; GetMessage must not be called.
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) {
			t.Fatal("GetMessage must not be called when local validation fails")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--stdin"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "exactly one of")
	})
}

func TestDraftCommand_ReplyAll_GetProfileFails(t *testing.T) {
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReply(), nil },
		GetProfileFunc: func(_ context.Context) (*gmailapi.Profile, error) {
			return nil, fmt.Errorf("profile boom")
		},
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			t.Fatal("CreateDraft must not be called when GetProfile fails for --reply-all")
			return nil, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "fetching profile")
	})
}

func TestDraftCommand_ReplyAll_EmptyProfileFallsBackToFrom(t *testing.T) {
	// Profile returns empty EmailAddress; --from alias supplies the self set.
	var seen gmailapi.DraftMessage
	src := srcReply()
	src.To = "alias@example.com, bob@example.com"
	mock := &MockGmailClient{
		GetMessageFunc:  func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		GetProfileFunc:  func(_ context.Context) (*gmailapi.Profile, error) { return &gmailapi.Profile{}, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) { seen = msg; return &gmailapi.DraftResult{ID: "d1"}, nil },
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--reply-all", "--from", "alias@example.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		// alias filtered via --from only; carol@example.com from source Cc; bob remains.
		testutil.LenSlice(t, len(seen.Cc), 2)
		testutil.Equal(t, seen.Cc[0], "bob@example.com")
		testutil.Equal(t, seen.Cc[1], "carol@example.com")
	})
}

func TestDraftCommand_ReplyTo_FirstReplyEmptyReferences(t *testing.T) {
	// First reply in a thread: source has no prior References — derived chain is just [Message-Id].
	var seen gmailapi.DraftMessage
	src := srcReply()
	src.References = ""
	mock := &MockGmailClient{
		GetMessageFunc:  func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) { seen = msg; return &gmailapi.DraftResult{ID: "d1"}, nil },
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.LenSlice(t, len(seen.References), 1)
		testutil.Equal(t, seen.References[0], "<orig@example.com>")
	})
}

func TestDraftCommand_ReplyTo_WithAttachment(t *testing.T) {
	// Reply mode + --attach must still propagate threading headers through the multipart path.
	tmp := t.TempDir()
	att := filepath.Join(tmp, "note.txt")
	if err := os.WriteFile(att, []byte("attached"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc:  func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReply(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) { seen = msg; return &gmailapi.DraftResult{ID: "d1"}, nil },
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain", "--attach", att})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.NoError(t, err)
		testutil.Equal(t, seen.ThreadID, "thread-src")
		testutil.Equal(t, seen.InReplyTo, "<orig@example.com>")
		testutil.LenSlice(t, len(seen.Attachments), 1)
	})
}

func TestDraftCommand_ReplyTo_RejectsHeaderInjection(t *testing.T) {
	mock := &MockGmailClient{}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src\r\nBcc: evil@x.com", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "CR or LF")
	})
}

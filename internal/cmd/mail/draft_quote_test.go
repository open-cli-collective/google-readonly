package mail

import (
	"context"
	"strings"
	"testing"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// --- Reply quoting tests (#125) ---

// srcReplyWithBody returns a reply source with a body and a Date whose offset
// is chosen so that converting to UTC changes the weekday, day, and hour — any
// accidental .UTC()/.Local() in the attribution is then caught.
//
// Source-local: Tue, May 12, 2026 at 11:30 PM (-0530).
// UTC would be:  Wed, May 13, 2026 at 5:00 AM.
func srcReplyWithBody() *gmailapi.Message {
	s := srcReply()
	s.Date = "Tue, 12 May 2026 23:30:00 -0530"
	s.Body = "Line one\n\nLine two"
	return s
}

const wantAttrib = "On Tue, May 12, 2026 at 11:30 PM Alice <alice@example.com> wrote:"

func TestDraftCommand_Reply_QuotesPlainByDefault(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "T27 reply", "--plain"})
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		want := "T27 reply\n\n" + wantAttrib + "\n\n> Line one\n>\n> Line two"
		testutil.Equal(t, string(seen.Body), want)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyPlainText)
		// No UTC drift: must not contain the UTC-converted day/hour.
		testutil.NotContains(t, string(seen.Body), "May 13")
		testutil.NotContains(t, string(seen.Body), "5:00 AM")
	})
}

func TestDraftCommand_Reply_QuotesHTMLWithGmailQuoteMarkup(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "T28 **reply**"}) // markdown -> HTML
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		body := string(seen.Body)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyHTML)
		testutil.Contains(t, body, "<strong>reply</strong>")
		testutil.Contains(t, body, `<div class="gmail_quote">`)
		testutil.Contains(t, body, `<blockquote class="gmail_quote" style="margin:0 0 0 .8ex;border-left:1px solid #ccc;padding-left:1ex">`)
		// Attribution is HTML-escaped: the raw <addr> angle brackets must not
		// leak as markup.
		testutil.Contains(t, body, "On Tue, May 12, 2026 at 11:30 PM Alice &lt;alice@example.com&gt; wrote:")
		testutil.NotContains(t, body, "<alice@example.com>")
		// Ordering: the quote follows the authored body and is immediately
		// preceded by the "\n" separator we add.
		authoredIdx := strings.Index(body, "<strong>reply</strong>")
		quoteStart := strings.Index(body, `<div class="gmail_quote">`)
		testutil.True(t, authoredIdx >= 0 && quoteStart > authoredIdx)
		testutil.Equal(t, body[quoteStart-1], byte('\n'))
	})
}

// CRLF must not leak into the HTML quote (no "\r" / "\r<br>").
func TestDraftCommand_Reply_QuotesHTMLCRLFSourceBody(t *testing.T) {
	src := srcReplyWithBody()
	src.Body = "Line one\r\n\r\nLine two"
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "hi"}) // HTML branch
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		body := string(seen.Body)
		testutil.NotContains(t, body, "\r")
		testutil.Contains(t, body, "Line one<br>")
		testutil.Contains(t, body, "Line two")
	})
}

// The HTML quote branch also fires for raw --html replies, not just markdown.
func TestDraftCommand_Reply_RawHTMLBodyStillQuotes(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--html", "--body", "<p>raw body</p>"})
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		body := string(seen.Body)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyHTML)
		testutil.True(t, strings.HasPrefix(body, "<p>raw body</p>")) // raw HTML passed verbatim
		testutil.Contains(t, body, `<div class="gmail_quote">`)
		testutil.Contains(t, body, "Line one<br>")
	})
}

func TestDraftCommand_Reply_HTMLEscapesMarkupInDisplayName(t *testing.T) {
	src := srcReplyWithBody()
	src.From = `"<script>alert(1)</script>" <alice@example.com>`
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return src, nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "hi"}) // HTML branch
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		body := string(seen.Body)
		testutil.NotContains(t, body, "<script>alert(1)</script>")
		testutil.Contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;")
	})
}

func TestDraftCommand_Reply_BareReplyIsQuoteOnly(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--plain"}) // zero body sources
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		// No authored body => no leading separator: body is exactly the quote.
		want := wantAttrib + "\n\n> Line one\n>\n> Line two"
		testutil.Equal(t, string(seen.Body), want)
	})
}

func TestDraftCommand_Reply_NoQuoteWithBody(t *testing.T) {
	var seen gmailapi.DraftMessage
	sawIncludeBody := true
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, includeBody bool) (*gmailapi.Message, error) {
			sawIncludeBody = includeBody
			return srcReplyWithBody(), nil
		},
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--no-quote", "--body", "T30", "--plain"})
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		testutil.Equal(t, string(seen.Body), "T30") // no attribution, no quote
		testutil.False(t, sawIncludeBody)           // --no-quote keeps metadata-only fetch
	})
}

func TestDraftCommand_Reply_DefaultFetchesBody(t *testing.T) {
	sawIncludeBody := false
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, includeBody bool) (*gmailapi.Message, error) {
			sawIncludeBody = includeBody
			return srcReplyWithBody(), nil
		},
		CreateDraftFunc: func(_ context.Context, _ gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--body", "x", "--plain"})
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		testutil.True(t, sawIncludeBody) // default reply fetches full body
	})
}

func TestDraftCommand_Reply_NoQuoteBareIsBlank(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src", "--no-quote", "--plain"})
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		testutil.Equal(t, string(seen.Body), "") // byte-empty, like a blank draft today
	})
}

func TestDraftCommand_NoQuote_RequiresReplyTo(t *testing.T) {
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--to", "a@x.com", "--subject", "Hi", "--body", "x", "--no-quote"})
	withMockClient(&MockGmailClient{}, func() {
		err := cmd.Execute()
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "--no-quote")
		testutil.Contains(t, err.Error(), "--reply-to")
	})
}

func TestDraftCommand_Reply_EmptySourceBodySkipsQuote(t *testing.T) {
	src := srcReply() // no Body
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
		testutil.NoError(t, cmd.Execute())
		testutil.Equal(t, string(seen.Body), "x") // no attribution appended
	})
}

func TestDraftCommand_Reply_DateParseFailureFallsBackToRaw(t *testing.T) {
	src := srcReplyWithBody()
	src.Date = "definitely not a date"
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
		testutil.NoError(t, cmd.Execute())
		testutil.Contains(t, string(seen.Body), "On definitely not a date Alice <alice@example.com> wrote:")
	})
}

// Real email text/plain bodies are CRLF. A CRLF blank line must quote as ">"
// (not "> \r") and content lines must not carry a trailing CR.
func TestDraftCommand_Reply_QuotesCRLFSourceBody(t *testing.T) {
	src := srcReplyWithBody()
	src.Body = "Line one\r\n\r\nLine two"
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
		testutil.NoError(t, cmd.Execute())
		want := "x\n\n" + wantAttrib + "\n\n> Line one\n>\n> Line two"
		testutil.Equal(t, string(seen.Body), want)
		testutil.NotContains(t, string(seen.Body), "\r")
	})
}

// The literal product path `gro mail draft --reply-to <id>` (no flags) defaults
// to markdown/HTML. A bare reply must have no leading newline and begin with
// the gmail_quote block.
func TestDraftCommand_Reply_BareReplyDefaultModeIsQuoteOnlyHTML(t *testing.T) {
	var seen gmailapi.DraftMessage
	mock := &MockGmailClient{
		GetMessageFunc: func(_ context.Context, _ string, _ bool) (*gmailapi.Message, error) { return srcReplyWithBody(), nil },
		CreateDraftFunc: func(_ context.Context, msg gmailapi.DraftMessage) (*gmailapi.DraftResult, error) {
			seen = msg
			return &gmailapi.DraftResult{ID: "d1"}, nil
		},
	}
	cmd := newDraftCommand()
	cmd.SetArgs([]string{"--reply-to", "msg-src"}) // no body source, no --plain
	withMockClient(mock, func() {
		testutil.NoError(t, cmd.Execute())
		body := string(seen.Body)
		testutil.Equal(t, seen.BodyKind, gmailapi.DraftBodyHTML)
		testutil.True(t, strings.HasPrefix(body, `<div class="gmail_quote">`))
		testutil.Contains(t, body, `<blockquote class="gmail_quote"`)
	})
}

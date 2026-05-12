package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"path/filepath"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// DraftBodyKind selects the Content-Type for the body part.
type DraftBodyKind int

const (
	// DraftBodyPlainText sends the body as text/plain; charset=UTF-8.
	// This is the zero value: a DraftMessage{} with an unset BodyKind
	// defaults to plain text, which is the safer interpretation when
	// callers construct DraftMessage directly without going through the CLI.
	DraftBodyPlainText DraftBodyKind = iota
	// DraftBodyHTML sends the body as text/html; charset=UTF-8.
	DraftBodyHTML
)

// DraftMessage describes a draft to be created. The command layer constructs
// this and hands it to (*Client).CreateDraft. The gmail package owns MIME
// assembly and the raw API call.
type DraftMessage struct {
	From        string   // optional; send-as alias
	To          []string // required; ≥1
	Cc          []string
	Bcc         []string
	Subject     string
	Body        []byte
	BodyKind    DraftBodyKind
	Attachments []DraftAttachment
}

// DraftAttachment is one attachment. Filename should be a basename; the gmail
// package re-applies filepath.Base defensively.
type DraftAttachment struct {
	Filename string
	MimeType string
	Data     []byte
}

// DraftResult is the response after creating a draft.
type DraftResult struct {
	ID        string `json:"id"`
	MessageID string `json:"messageId"`
	ThreadID  string `json:"threadId,omitempty"`
}

// CreateDraft assembles a MIME message and POSTs to users.drafts.create.
// The CLI never calls drafts.send — drafts sit in the user's Drafts folder
// for explicit human review.
func (c *Client) CreateDraft(ctx context.Context, msg DraftMessage) (*DraftResult, error) {
	raw, err := buildMIME(msg)
	if err != nil {
		return nil, fmt.Errorf("building draft MIME: %w", err)
	}

	draft := &gmailapi.Draft{
		Message: &gmailapi.Message{
			Raw: base64.URLEncoding.EncodeToString(raw),
		},
	}
	resp, err := c.service.Users.Drafts.Create(c.userID, draft).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating draft: %w", err)
	}
	out := &DraftResult{ID: resp.Id}
	if resp.Message != nil {
		out.MessageID = resp.Message.Id
		out.ThreadID = resp.Message.ThreadId
	}
	return out, nil
}

// buildMIME assembles an RFC 5322 message with CRLF line endings and
// MIME-Version: 1.0, suitable for base64url-encoding into Gmail's
// Message.Raw field.
func buildMIME(msg DraftMessage) ([]byte, error) {
	if len(msg.To) == 0 {
		return nil, fmt.Errorf("draft has no To recipients")
	}
	if err := guardHeaderInjection(msg); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("MIME-Version: 1.0\r\n")

	if msg.From != "" {
		fmt.Fprintf(&buf, "From: %s\r\n", (&mail.Address{Address: msg.From}).String())
	}
	if h := addressHeader("To", msg.To); h != "" {
		buf.WriteString(h)
	}
	if h := addressHeader("Cc", msg.Cc); h != "" {
		buf.WriteString(h)
	}
	if h := addressHeader("Bcc", msg.Bcc); h != "" {
		buf.WriteString(h)
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", msg.Subject))

	if len(msg.Attachments) == 0 {
		// Single-part: emit content headers and base64 body inline.
		buf.WriteString(bodyContentType(msg.BodyKind))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString("\r\n")
		buf.Write(wrapBase64(msg.Body))
		return buf.Bytes(), nil
	}

	// Multipart/mixed.
	// Order matters: NewWriter is created first to obtain a boundary, but
	// the top-level Content-Type header and blank line MUST be written to
	// buf BEFORE any CreatePart call — CreatePart writes the first boundary
	// delimiter, which must come after the blank line that terminates the
	// message headers. Do not reorder.
	mw := multipart.NewWriter(&buf)
	ct := mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": mw.Boundary()})
	buf.WriteString("Content-Type: " + ct + "\r\n\r\n")

	// Body part.
	bodyHdr := textproto.MIMEHeader{}
	bodyHdr.Set("Content-Type", bodyContentTypeValue(msg.BodyKind))
	bodyHdr.Set("Content-Transfer-Encoding", "base64")
	bw, err := mw.CreatePart(bodyHdr)
	if err != nil {
		return nil, fmt.Errorf("creating body part: %w", err)
	}
	if _, err := bw.Write(wrapBase64(msg.Body)); err != nil {
		return nil, fmt.Errorf("writing body part: %w", err)
	}

	// Attachment parts.
	for i, att := range msg.Attachments {
		basename := filepath.Base(att.Filename)
		ct, err := buildAttachmentContentType(att.MimeType, basename)
		if err != nil {
			return nil, fmt.Errorf("attachment %d content-type: %w", i, err)
		}
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Type", ct)
		hdr.Set("Content-Transfer-Encoding", "base64")
		hdr.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": basename}))
		aw, err := mw.CreatePart(hdr)
		if err != nil {
			return nil, fmt.Errorf("creating attachment part %d: %w", i, err)
		}
		if _, err := aw.Write(wrapBase64(att.Data)); err != nil {
			return nil, fmt.Errorf("writing attachment %d: %w", i, err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}
	return buf.Bytes(), nil
}

// guardHeaderInjection rejects CR/LF in any field that becomes a header or
// header parameter, defending the raw-MIME boundary independently of the
// command-layer validator.
func guardHeaderInjection(msg DraftMessage) error {
	check := func(field, val string) error {
		if strings.ContainsAny(val, "\r\n") {
			return fmt.Errorf("invalid %s: contains CR or LF", field)
		}
		return nil
	}
	if err := check("subject", msg.Subject); err != nil {
		return err
	}
	if err := check("from", msg.From); err != nil {
		return err
	}
	for _, a := range msg.To {
		if err := check("to", a); err != nil {
			return err
		}
	}
	for _, a := range msg.Cc {
		if err := check("cc", a); err != nil {
			return err
		}
	}
	for _, a := range msg.Bcc {
		if err := check("bcc", a); err != nil {
			return err
		}
	}
	for _, att := range msg.Attachments {
		if err := check("attachment filename", att.Filename); err != nil {
			return err
		}
	}
	return nil
}

func addressHeader(name string, addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}
	formatted := make([]string, len(addrs))
	for i, a := range addrs {
		formatted[i] = (&mail.Address{Address: a}).String()
	}
	return fmt.Sprintf("%s: %s\r\n", name, strings.Join(formatted, ", "))
}

func bodyContentType(k DraftBodyKind) string {
	return "Content-Type: " + bodyContentTypeValue(k) + "\r\n"
}

func bodyContentTypeValue(k DraftBodyKind) string {
	if k == DraftBodyHTML {
		return "text/html; charset=UTF-8"
	}
	return "text/plain; charset=UTF-8"
}

// buildAttachmentContentType parses any params in the supplied MIME type
// (e.g. "text/plain; charset=utf-8"), merges a name=<basename> param, and
// re-formats. Falls back to application/octet-stream if mt is empty.
func buildAttachmentContentType(mt, basename string) (string, error) {
	if mt == "" {
		mt = "application/octet-stream"
	}
	mediaType, params, err := mime.ParseMediaType(mt)
	if err != nil {
		return "", fmt.Errorf("parsing media type %q: %w", mt, err)
	}
	if params == nil {
		params = map[string]string{}
	}
	params["name"] = basename
	return mime.FormatMediaType(mediaType, params), nil
}

// wrapBase64 standard-encodes b and wraps the output at 76 columns with CRLF,
// per RFC 2045 §6.8.
func wrapBase64(b []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(b)
	var out bytes.Buffer
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		out.WriteString(encoded[i:end])
		out.WriteString("\r\n")
	}
	return out.Bytes()
}

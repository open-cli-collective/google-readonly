package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// mimePart is a snapshot of a multipart.Part with its body already read.
// Iterating multipart.NextPart consumes the previous part's body, so we
// read each part eagerly and store the bytes here.
type mimePart struct {
	Header textproto.MIMEHeader
	Body   []byte // raw body bytes, before CTE decoding
}

// parseMIME decodes the raw MIME bytes produced by buildMIME and returns the
// parsed top-level message plus the body bytes for single-part messages, or
// the slice of parts for multipart/mixed.
func parseMIME(t *testing.T, raw []byte) (*mail.Message, string, []byte, []mimePart) {
	t.Helper()
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("mail.ReadMessage: %v", err)
	}

	ct := m.Header.Get("Content-Type")
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("ParseMediaType(%q): %v", ct, err)
	}

	if !strings.HasPrefix(mt, "multipart/") {
		body, err := io.ReadAll(m.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if cte := m.Header.Get("Content-Transfer-Encoding"); strings.EqualFold(cte, "base64") {
			compact := strings.ReplaceAll(strings.ReplaceAll(string(body), "\r\n", ""), "\n", "")
			decoded, err := base64.StdEncoding.DecodeString(compact)
			if err != nil {
				t.Fatalf("decode body base64: %v", err)
			}
			body = decoded
		}
		return m, mt, body, nil
	}

	r := multipart.NewReader(m.Body, params["boundary"])
	var parts []mimePart
	for {
		p, err := r.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		body, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("read part body: %v", err)
		}
		parts = append(parts, mimePart{Header: p.Header, Body: body})
	}
	return m, mt, nil, parts
}

func decodePart(t *testing.T, p mimePart) []byte {
	t.Helper()
	if cte := p.Header.Get("Content-Transfer-Encoding"); strings.EqualFold(cte, "base64") {
		compact := strings.ReplaceAll(strings.ReplaceAll(string(p.Body), "\r\n", ""), "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(compact)
		if err != nil {
			t.Fatalf("decode part base64: %v", err)
		}
		return decoded
	}
	return p.Body
}

func TestBuildMIME_PlainText(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"alice@example.com"},
		Subject:  "Hi",
		Body:     []byte("hello world"),
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	if !strings.Contains(string(raw), "MIME-Version: 1.0\r\n") {
		t.Errorf("missing MIME-Version header")
	}
	if hasBareLF(string(raw)) {
		t.Errorf("output contains bare LF (must be CRLF)")
	}

	m, mt, body, parts := parseMIME(t, raw)
	if got := m.Header.Get("To"); got != "<alice@example.com>" {
		t.Errorf("To = %q, want %q", got, "<alice@example.com>")
	}
	if got := m.Header.Get("Subject"); got != "Hi" {
		t.Errorf("Subject = %q, want %q", got, "Hi")
	}
	if mt != "text/plain" {
		t.Errorf("Content-Type media = %q, want text/plain", mt)
	}
	if parts != nil {
		t.Errorf("expected single-part, got multipart with %d parts", len(parts))
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}

func TestBuildMIME_HTML(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"alice@example.com"},
		Subject:  "Hi",
		Body:     []byte("<p>hello</p>"),
		BodyKind: DraftBodyHTML,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, mt, body, _ := parseMIME(t, raw)
	if mt != "text/html" {
		t.Errorf("Content-Type media = %q, want text/html", mt)
	}
	if string(body) != "<p>hello</p>" {
		t.Errorf("body = %q, want %q", string(body), "<p>hello</p>")
	}
}

func TestBuildMIME_MultipleRecipients(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com", "b@x.com"},
		Cc:       []string{"c@x.com"},
		Bcc:      []string{"d@x.com"},
		Subject:  "Hi",
		Body:     []byte("text"),
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	m, _, _, _ := parseMIME(t, raw)
	to, err := mail.ParseAddressList(m.Header.Get("To"))
	if err != nil || len(to) != 2 {
		t.Errorf("To parse: err=%v len=%d", err, len(to))
	}
	cc, err := mail.ParseAddressList(m.Header.Get("Cc"))
	if err != nil || len(cc) != 1 {
		t.Errorf("Cc parse: err=%v len=%d", err, len(cc))
	}
	bcc, err := mail.ParseAddressList(m.Header.Get("Bcc"))
	if err != nil || len(bcc) != 1 {
		t.Errorf("Bcc parse: err=%v len=%d", err, len(bcc))
	}
}

func TestBuildMIME_From(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		From:     "work@me.com",
		To:       []string{"a@x.com"},
		Subject:  "Hi",
		Body:     []byte("text"),
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	m, _, _, _ := parseMIME(t, raw)
	if got := m.Header.Get("From"); got != "<work@me.com>" {
		t.Errorf("From = %q, want %q", got, "<work@me.com>")
	}
}

func TestBuildMIME_NonASCIISubject(t *testing.T) {
	t.Parallel()
	want := "Café meeting — Q4 ☕"
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  want,
		Body:     []byte("text"),
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	m, _, _, _ := parseMIME(t, raw)
	dec := new(mime.WordDecoder)
	got, err := dec.DecodeHeader(m.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
}

func TestBuildMIME_SingleAttachment(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Report",
		Body:     []byte("see attached"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "report.pdf", MimeType: "application/pdf", Data: []byte("%PDF-1.4 fake")},
		},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, mt, body, parts := parseMIME(t, raw)
	if mt != "multipart/mixed" {
		t.Errorf("media = %q, want multipart/mixed", mt)
	}
	if body != nil {
		t.Errorf("expected nil top-level body for multipart, got %d bytes", len(body))
	}
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	if got := string(decodePart(t, parts[0])); got != "see attached" {
		t.Errorf("body part = %q, want %q", got, "see attached")
	}
	if got := string(decodePart(t, parts[1])); got != "%PDF-1.4 fake" {
		t.Errorf("attachment bytes = %q", got)
	}
	cd := parts[1].Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		t.Fatalf("parse CD: %v", err)
	}
	if params["filename"] != "report.pdf" {
		t.Errorf("filename = %q, want report.pdf", params["filename"])
	}
}

func TestBuildMIME_MultipleAttachments_MixedTypes(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Bundle",
		Body:     []byte("two files"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "report.pdf", MimeType: "application/pdf", Data: []byte("PDF bytes")},
			{Filename: "data.csv", MimeType: "text/csv; charset=utf-8", Data: []byte("a,b,c\n1,2,3")},
		},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, _, _, parts := parseMIME(t, raw)
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3", len(parts))
	}
	if got := string(decodePart(t, parts[1])); got != "PDF bytes" {
		t.Errorf("pdf bytes = %q", got)
	}
	if got := string(decodePart(t, parts[2])); got != "a,b,c\n1,2,3" {
		t.Errorf("csv bytes = %q", got)
	}
	mt, params, err := mime.ParseMediaType(parts[2].Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse CSV CT: %v", err)
	}
	if mt != "text/csv" || params["charset"] != "utf-8" || params["name"] != "data.csv" {
		t.Errorf("csv content-type wrong: mt=%q params=%v", mt, params)
	}
}

func TestBuildMIME_TextPlainCharset(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Plain",
		Body:     []byte("body"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "notes.txt", MimeType: "text/plain; charset=utf-8", Data: []byte("plain notes")},
		},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, _, _, parts := parseMIME(t, raw)
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	mt, params, err := mime.ParseMediaType(parts[1].Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse CT: %v", err)
	}
	if mt != "text/plain" || params["charset"] != "utf-8" || params["name"] != "notes.txt" {
		t.Errorf("txt content-type wrong: mt=%q params=%v", mt, params)
	}
}

func TestBuildMIME_NonASCIIFilename(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "i18n",
		Body:     []byte("body"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "résumé.pdf", MimeType: "application/pdf", Data: []byte("PDF")},
		},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, _, _, parts := parseMIME(t, raw)
	_, params, err := mime.ParseMediaType(parts[1].Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("parse CD: %v", err)
	}
	if params["filename"] != "résumé.pdf" {
		t.Errorf("filename = %q, want résumé.pdf", params["filename"])
	}
}

func TestBuildMIME_BasenameOnly_NoLocalPathLeak(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Defensive",
		Body:     []byte("body"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "/tmp/secret/foo.pdf", MimeType: "application/pdf", Data: []byte("PDF")},
		},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	if strings.Contains(string(raw), "/tmp/secret") {
		t.Errorf("raw MIME leaked path: contains /tmp/secret")
	}
	_, _, _, parts := parseMIME(t, raw)
	_, params, err := mime.ParseMediaType(parts[1].Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("parse CD: %v", err)
	}
	if params["filename"] != "foo.pdf" {
		t.Errorf("filename = %q, want foo.pdf (basename)", params["filename"])
	}
}

func TestBuildMIME_Base64PayloadLineLength(t *testing.T) {
	t.Parallel()
	big := make([]byte, 4096)
	for i := range big {
		big[i] = 'x'
	}
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Big",
		Body:     big,
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	parts := strings.SplitN(string(raw), "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected header/body split, got %d parts", len(parts))
	}
	for _, line := range strings.Split(parts[1], "\r\n") {
		if len(line) > 76 {
			t.Errorf("base64 line exceeds 76 chars: len=%d", len(line))
		}
	}
}

func TestBuildMIME_RejectsHeaderInjection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mut  func(*DraftMessage)
	}{
		{"subject CRLF", func(m *DraftMessage) { m.Subject = "evil\r\nBcc: x@y.com" }},
		{"subject LF", func(m *DraftMessage) { m.Subject = "evil\nBcc: x@y.com" }},
		{"to CRLF", func(m *DraftMessage) { m.To = []string{"a@x.com\r\nBcc: evil@x.com"} }},
		{"from CRLF", func(m *DraftMessage) { m.From = "a@x.com\r\nBcc: evil@x.com" }},
		{"cc CRLF", func(m *DraftMessage) { m.Cc = []string{"a@x.com\r\nBcc: evil@x.com"} }},
		{"bcc CRLF", func(m *DraftMessage) { m.Bcc = []string{"a@x.com\r\nBcc: evil@x.com"} }},
		{"attachment filename LF", func(m *DraftMessage) {
			m.Attachments = []DraftAttachment{{Filename: "a\nb.pdf", MimeType: "application/pdf", Data: []byte("x")}}
		}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := DraftMessage{
				To:       []string{"a@x.com"},
				Subject:  "ok",
				Body:     []byte("body"),
				BodyKind: DraftBodyPlainText,
			}
			tc.mut(&msg)
			if _, err := buildMIME(msg); err == nil {
				t.Errorf("expected error for injection case, got nil")
			}
		})
	}
}

// --- API wiring tests ---

func TestCreateDraft_APIWiring_SinglePart(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Hi",
		Body:     []byte("hello"),
		BodyKind: DraftBodyPlainText,
	}
	wantRaw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}

	var gotMethod, gotPath, gotRawDecoded string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var body struct {
			Message struct {
				Raw string `json:"raw"`
			} `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		decoded, derr := base64.URLEncoding.DecodeString(body.Message.Raw)
		if derr != nil {
			decoded, derr = base64.RawURLEncoding.DecodeString(body.Message.Raw)
		}
		if derr != nil {
			t.Errorf("decode raw: %v", derr)
		}
		gotRawDecoded = string(decoded)
		resp := &gmailapi.Draft{
			Id: "d-123",
			Message: &gmailapi.Message{
				Id:       "m-456",
				ThreadId: "t-789",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ctx := context.Background()
	svc, err := gmailapi.NewService(ctx,
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	c := &Client{service: svc, userID: "me"}
	result, err := c.CreateDraft(ctx, msg)
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/gmail/v1/users/me/drafts" {
		t.Errorf("path = %q, want /gmail/v1/users/me/drafts", gotPath)
	}
	if gotRawDecoded != string(wantRaw) {
		t.Errorf("raw payload mismatch\ngot:  %q\nwant: %q", gotRawDecoded, string(wantRaw))
	}
	if result.ID != "d-123" || result.MessageID != "m-456" || result.ThreadID != "t-789" {
		t.Errorf("result = %+v", result)
	}
}

func TestCreateDraft_APIWiring_Multipart_Structural(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "WithAtt",
		Body:     []byte("body"),
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{
			{Filename: "report.pdf", MimeType: "application/pdf", Data: []byte("PDFBYTES")},
		},
	}

	var seenRaw []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message struct {
				Raw string `json:"raw"`
			} `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		decoded, derr := base64.URLEncoding.DecodeString(body.Message.Raw)
		if derr != nil {
			decoded, _ = base64.RawURLEncoding.DecodeString(body.Message.Raw)
		}
		seenRaw = decoded
		resp := &gmailapi.Draft{Id: "d-1", Message: &gmailapi.Message{Id: "m-1"}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ctx := context.Background()
	svc, err := gmailapi.NewService(ctx,
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	c := &Client{service: svc, userID: "me"}
	if _, err := c.CreateDraft(ctx, msg); err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	_, mt, _, parts := parseMIME(t, seenRaw)
	if mt != "multipart/mixed" {
		t.Errorf("media = %q, want multipart/mixed", mt)
	}
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	if got := string(decodePart(t, parts[1])); got != "PDFBYTES" {
		t.Errorf("att bytes = %q", got)
	}
}

func TestBuildMIME_RejectsEmptyTo(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		Subject:  "Hi",
		Body:     []byte("body"),
		BodyKind: DraftBodyPlainText,
	}
	if _, err := buildMIME(msg); err == nil {
		t.Errorf("expected error for empty To, got nil")
	}
}

func TestBuildMIME_EmptyBody(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Hi",
		Body:     []byte{},
		BodyKind: DraftBodyPlainText,
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, _, body, _ := parseMIME(t, raw)
	if len(body) != 0 {
		t.Errorf("decoded body = %q, want empty", body)
	}
}

func TestBuildMIME_MultipartBase64LineLength(t *testing.T) {
	t.Parallel()
	big := make([]byte, 4096)
	for i := range big {
		big[i] = 'x'
	}
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Big",
		Body:     big,
		BodyKind: DraftBodyPlainText,
		Attachments: []DraftAttachment{{
			Filename: "blob.bin",
			MimeType: "application/octet-stream",
			Data:     big,
		}},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, _, _, parts := parseMIME(t, raw)
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	for i, p := range parts {
		for _, line := range strings.Split(string(p.Body), "\r\n") {
			if len(line) > 76 {
				t.Errorf("part %d base64 line exceeds 76 chars: len=%d", i, len(line))
			}
		}
	}
}

func TestBuildMIME_MultipartHTMLBody(t *testing.T) {
	t.Parallel()
	msg := DraftMessage{
		To:       []string{"a@x.com"},
		Subject:  "Hi",
		Body:     []byte("<p>hello</p>"),
		BodyKind: DraftBodyHTML,
		Attachments: []DraftAttachment{{
			Filename: "note.txt",
			MimeType: "text/plain",
			Data:     []byte("attached"),
		}},
	}
	raw, err := buildMIME(msg)
	if err != nil {
		t.Fatalf("buildMIME: %v", err)
	}
	_, mt, _, parts := parseMIME(t, raw)
	if mt != "multipart/mixed" {
		t.Errorf("media = %q, want multipart/mixed", mt)
	}
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want 2", len(parts))
	}
	bodyCT, _, err := mime.ParseMediaType(parts[0].Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse body part Content-Type: %v", err)
	}
	if bodyCT != "text/html" {
		t.Errorf("body part media = %q, want text/html", bodyCT)
	}
	if got := string(decodePart(t, parts[0])); got != "<p>hello</p>" {
		t.Errorf("body decoded = %q", got)
	}
}

func hasBareLF(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			return true
		}
	}
	return false
}

package mail

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	gmailapi "github.com/open-cli-collective/google-readonly/internal/gmail"
)

func newDraftCommand() *cobra.Command {
	var (
		to       string
		cc       string
		bcc      string
		fromAddr string
		subject  string
		body     string
		file     string
		stdin    bool
		plain    bool
		htmlMode bool
		attach   []string
		jsonOut  bool
		replyTo  string
		replyAll bool
	)

	cmd := &cobra.Command{
		Use:   "draft",
		Args:  cobra.NoArgs,
		Short: "Compose a Gmail draft (never sent automatically)",
		Long: `Compose a Gmail draft and save it to the Drafts folder for human review.

The CLI never calls drafts.send — sending requires explicit action in Gmail.

Body input is markdown by default and rendered to HTML. Use --plain for plain
text, --html to send raw HTML verbatim. Body source is one of --body, --stdin,
or --file.

Examples:
  gro mail draft --to alice@example.com --subject "Hi" --body "**hello**"
  gro mail draft --to "a@x.com, b@x.com" --subject "Sync" --file notes.md
  echo "# Hello" | gro mail draft --to a@x.com --subject "Hi" --stdin
  gro mail draft --to a@x.com --subject "Plain" --body "no md" --plain
  gro mail draft --to a@x.com --subject "Report" --body "see attached" --attach report.pdf
  gro mail draft --reply-to <message-id> --body "thanks, will review"
  gro mail draft --reply-to <message-id> --reply-all --body "..."`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// --reply-to with an empty value is a user error (often a shell
			// variable that didn't expand). Reject it explicitly so we don't
			// silently produce an unthreaded draft.
			if cmd.Flags().Changed("reply-to") && strings.TrimSpace(replyTo) == "" {
				return fmt.Errorf("--reply-to requires a non-empty message ID")
			}
			isReply := cmd.Flags().Changed("reply-to")

			// 1. Required-flag checks. In reply mode, --to and --subject are
			//    derived from the source message and only required if not derivable.
			if replyAll && !isReply {
				return fmt.Errorf("--reply-all requires --reply-to")
			}
			if !isReply {
				if !cmd.Flags().Changed("to") {
					return fmt.Errorf("--to is required")
				}
				if !cmd.Flags().Changed("subject") {
					return fmt.Errorf("--subject is required (use --subject \"\" for empty subject)")
				}
			}

			// 2. Header-injection guard on raw flag values.
			for _, f := range []struct{ name, val string }{
				{"--to", to}, {"--cc", cc}, {"--bcc", bcc},
				{"--from", fromAddr}, {"--subject", subject},
				{"--reply-to", replyTo},
			} {
				if strings.ContainsAny(f.val, "\r\n") {
					return fmt.Errorf("%s contains illegal CR or LF", f.name)
				}
			}

			// 3. Address parsing on user-supplied flags.
			toAddrs, err := parseAddressList("--to", to)
			if err != nil {
				return err
			}
			ccAddrs, err := parseAddressList("--cc", cc)
			if err != nil {
				return err
			}
			bccAddrs, err := parseAddressList("--bcc", bcc)
			if err != nil {
				return err
			}
			if fromAddr != "" {
				parsed, err := mail.ParseAddress(fromAddr)
				if err != nil {
					return fmt.Errorf("--from is not a valid email address: %w", err)
				}
				// Normalise to the bare address so display-name input like
				// "Work <work@me.com>" is reduced to "work@me.com" before
				// reaching the MIME builder, matching how --to/--cc/--bcc
				// are handled.
				fromAddr = parsed.Address
			}

			// 3b. Local recipient validation (no I/O yet).
			//     - Non-reply mode: --to is required and must parse to ≥1 address.
			//     - Reply mode: an explicit --to override must still parse to ≥1
			//       address; otherwise we defer the check until after derivation.
			if !isReply && len(toAddrs) == 0 {
				return fmt.Errorf("--to must contain at least one address")
			}
			if isReply && cmd.Flags().Changed("to") && len(toAddrs) == 0 {
				return fmt.Errorf("--to must contain at least one address")
			}

			// 4. Body source: exactly one of --body, --stdin, --file.
			bodySources := 0
			if cmd.Flags().Changed("body") {
				bodySources++
			}
			if stdin {
				bodySources++
			}
			if cmd.Flags().Changed("file") {
				bodySources++
			}
			if bodySources != 1 {
				return fmt.Errorf("exactly one of --body, --stdin, or --file is required")
			}

			// 5. Format mode: at most one of --plain or --html.
			if plain && htmlMode {
				return fmt.Errorf("--plain and --html are mutually exclusive")
			}

			// 6. Resolve body bytes.
			var bodyBytes []byte
			switch {
			case stdin:
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("reading body from stdin: %w", err)
				}
				bodyBytes = b
			case cmd.Flags().Changed("file"):
				b, err := os.ReadFile(file) // #nosec G304 -- file path is intentionally user-supplied via --file
				if err != nil {
					return fmt.Errorf("reading body file: %w", err)
				}
				bodyBytes = b
			default:
				bodyBytes = []byte(body)
			}

			// 7. Render or pass through.
			var (
				outBody []byte
				kind    gmailapi.DraftBodyKind
			)
			switch {
			case plain:
				outBody, kind = bodyBytes, gmailapi.DraftBodyPlainText
			case htmlMode:
				outBody, kind = bodyBytes, gmailapi.DraftBodyHTML
			default:
				rendered, err := renderMarkdown(bodyBytes)
				if err != nil {
					return fmt.Errorf("rendering markdown: %w", err)
				}
				outBody, kind = rendered, gmailapi.DraftBodyHTML
			}

			// 8. Load attachments.
			attachments := make([]gmailapi.DraftAttachment, 0, len(attach))
			for _, path := range attach {
				data, err := os.ReadFile(path) // #nosec G304 -- attachment path is intentionally user-supplied via --attach
				if err != nil {
					return fmt.Errorf("reading attachment %s: %w", path, err)
				}
				attachments = append(attachments, gmailapi.DraftAttachment{
					Filename: filepath.Base(path),
					MimeType: detectMimeType(path),
					Data:     data,
				})
			}

			// 9. Build client; in reply mode, fetch source + (optionally) profile and derive defaults.
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			var (
				threadID   string
				inReplyTo  string
				references []string
			)
			if isReply {
				src, err := client.GetMessage(cmd.Context(), replyTo, false)
				if err != nil {
					return fmt.Errorf("fetching reply source %s: %w", replyTo, err)
				}
				needs := replyNeeds{
					To:      !cmd.Flags().Changed("to"),
					Cc:      replyAll && !cmd.Flags().Changed("cc"),
					Subject: !cmd.Flags().Changed("subject"),
				}
				selfSet := map[string]bool{}
				if needs.Cc {
					profile, err := client.GetProfile(cmd.Context())
					if err != nil {
						return fmt.Errorf("fetching profile for --reply-all: %w", err)
					}
					if profile != nil && profile.EmailAddress != "" {
						selfSet[strings.ToLower(profile.EmailAddress)] = true
					}
					if fromAddr != "" {
						selfSet[strings.ToLower(fromAddr)] = true
					}
				}
				derived, err := deriveReplyDefaults(src, needs, selfSet)
				if err != nil {
					return err
				}
				threadID = derived.ThreadID
				inReplyTo = derived.InReplyTo
				references = derived.References
				if needs.To {
					toAddrs = derived.To
				}
				if needs.Cc {
					ccAddrs = derived.Cc
				}
				if needs.Subject {
					subject = derived.Subject
				}
			}

			// 9b. Post-derivation resolved-recipient validation.
			if len(toAddrs) == 0 {
				return fmt.Errorf("--to must contain at least one address")
			}

			result, err := client.CreateDraft(cmd.Context(), gmailapi.DraftMessage{
				From:        fromAddr,
				To:          toAddrs,
				Cc:          ccAddrs,
				Bcc:         bccAddrs,
				Subject:     subject,
				Body:        outBody,
				BodyKind:    kind,
				Attachments: attachments,
				ThreadID:    threadID,
				InReplyTo:   inReplyTo,
				References:  references,
			})
			if err != nil {
				return fmt.Errorf("creating draft: %w", err)
			}
			if result == nil {
				return fmt.Errorf("creating draft: empty response")
			}

			// 10. Print result.
			if jsonOut {
				return printJSON(result)
			}
			fmt.Printf("Draft created: %s\n", result.ID)
			fmt.Printf("To: %s\n", SanitizeOutput(strings.Join(toAddrs, ", ")))
			if len(ccAddrs) > 0 {
				fmt.Printf("Cc: %s\n", SanitizeOutput(strings.Join(ccAddrs, ", ")))
			}
			if len(bccAddrs) > 0 {
				fmt.Printf("Bcc: %s\n", SanitizeOutput(strings.Join(bccAddrs, ", ")))
			}
			fmt.Printf("Subject: %s\n", SanitizeOutput(subject))
			if len(attachments) > 0 {
				fmt.Printf("Attachments: %d\n", len(attachments))
				for _, a := range attachments {
					fmt.Printf("  - %s\n", SanitizeOutput(a.Filename))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient(s), comma-separated (display names are stripped; edit the draft in Gmail to set them)")
	cmd.Flags().StringVar(&cc, "cc", "", "Cc recipient(s), comma-separated (display names stripped)")
	cmd.Flags().StringVar(&bcc, "bcc", "", "Bcc recipient(s), comma-separated (display names stripped)")
	cmd.Flags().StringVar(&fromAddr, "from", "", "From address (Gmail send-as alias)")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Subject line")
	cmd.Flags().StringVar(&body, "body", "", "Body content (markdown by default)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Read body from file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read body from stdin")
	cmd.Flags().BoolVar(&plain, "plain", false, "Treat body as plain text (no markdown rendering)")
	cmd.Flags().BoolVar(&htmlMode, "html", false, "Treat body as raw HTML (no markdown rendering)")
	cmd.Flags().StringArrayVarP(&attach, "attach", "a", nil, "File path to attach (repeat for multiple)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output result as JSON")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Source Gmail message ID to reply to (derives To/Subject/threading)")
	cmd.Flags().BoolVar(&replyAll, "reply-all", false, "Include the source To/Cc as Cc on the reply (requires --reply-to)")

	return cmd
}

// replyDerivation is the result of analysing the source message for reply mode.
type replyDerivation struct {
	ThreadID   string
	InReplyTo  string
	References []string
	To         []string
	Cc         []string
	Subject    string
}

// replyNeeds tells deriveReplyDefaults which slots the caller will actually
// consume. Source headers are only parsed for needed slots, so an explicit
// --to/--cc/--subject override is unaffected by a malformed/missing source
// header for that slot.
type replyNeeds struct {
	To      bool
	Cc      bool
	Subject bool
}

func deriveReplyDefaults(src *gmailapi.Message, needs replyNeeds, selfSet map[string]bool) (replyDerivation, error) {
	if src == nil {
		return replyDerivation{}, fmt.Errorf("source message is nil")
	}
	// Message-Id is always required: it feeds In-Reply-To and References,
	// which are emitted regardless of override flags.
	if strings.TrimSpace(src.RFCMessageID) == "" {
		return replyDerivation{}, fmt.Errorf("source message %s has no Message-Id header", src.ID)
	}
	out := replyDerivation{
		ThreadID:   src.ThreadID,
		InReplyTo:  src.RFCMessageID,
		References: buildReferences(src.References, src.RFCMessageID),
	}
	if needs.To {
		if strings.TrimSpace(src.From) == "" {
			return replyDerivation{}, fmt.Errorf("source message %s has no From header", src.ID)
		}
		fromAddr, err := mail.ParseAddress(src.From)
		if err != nil {
			return replyDerivation{}, fmt.Errorf("parsing source From header %q: %w", src.From, err)
		}
		out.To = []string{fromAddr.Address}
	}
	if needs.Subject {
		out.Subject = addRePrefix(src.Subject)
	}
	if needs.Cc {
		ccAddrs, err := splitAddressHeaders(src.To, src.Cc)
		if err != nil {
			return replyDerivation{}, fmt.Errorf("parsing source To/Cc headers: %w", err)
		}
		out.Cc = filterSelf(ccAddrs, selfSet)
	}
	return out, nil
}

// buildReferences appends the source Message-Id to the existing References
// chain, splitting the raw header on whitespace (RFC 5322 §3.6.4). Returns
// the new chain with empty tokens dropped.
func buildReferences(rawRefs, msgID string) []string {
	out := make([]string, 0)
	for _, f := range strings.Fields(rawRefs) {
		if f != "" {
			out = append(out, f)
		}
	}
	out = append(out, msgID)
	return out
}

// splitAddressHeaders parses each non-empty raw header value via
// mail.ParseAddressList and concatenates the bare addresses. Returns an
// error on the first malformed header rather than silently dropping content.
func splitAddressHeaders(values ...string) ([]string, error) {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		addrs, err := mail.ParseAddressList(v)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", v, err)
		}
		for _, a := range addrs {
			out = append(out, a.Address)
		}
	}
	return out, nil
}

// filterSelf removes addresses present in selfSet (case-insensitive lookup)
// and de-duplicates while preserving first-seen order.
func filterSelf(addrs []string, selfSet map[string]bool) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		k := strings.ToLower(a)
		if selfSet[k] || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, a)
	}
	return out
}

// addRePrefix prefixes "Re: " unless the subject already begins with a
// case-insensitive "re:" (followed by space or end-of-string). Locale
// variants (Aw, Sv, etc.) are intentionally not recognised — Gmail itself
// doesn't normalise them.
func addRePrefix(subject string) string {
	trimmed := strings.TrimLeft(subject, " \t")
	if len(trimmed) >= 3 && strings.EqualFold(trimmed[:3], "re:") {
		return subject
	}
	return "Re: " + subject
}

// parseAddressList parses a comma-separated address list. An empty input is OK
// and returns nil — the caller decides whether emptiness is an error.
func parseAddressList(flag, raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	addrs, err := mail.ParseAddressList(raw)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid address list: %w", flag, err)
	}
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.Address
	}
	return out, nil
}

// detectMimeType resolves the MIME type for a file path via mime.TypeByExtension.
// Falls back to application/octet-stream.
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "application/octet-stream"
	}
	if mt := mime.TypeByExtension(ext); mt != "" {
		return mt
	}
	return "application/octet-stream"
}

// renderMarkdown converts markdown to HTML using goldmark with table,
// strikethrough, and task-list extensions enabled.
func renderMarkdown(src []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

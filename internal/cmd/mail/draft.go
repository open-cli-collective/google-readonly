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
  gro mail draft --to a@x.com --subject "Report" --body "see attached" --attach report.pdf`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 1. Required-flag checks.
			if !cmd.Flags().Changed("to") {
				return fmt.Errorf("--to is required")
			}
			if !cmd.Flags().Changed("subject") {
				return fmt.Errorf("--subject is required (use --subject \"\" for empty subject)")
			}

			// 2. Header-injection guard on raw flag values.
			for _, f := range []struct{ name, val string }{
				{"--to", to}, {"--cc", cc}, {"--bcc", bcc},
				{"--from", fromAddr}, {"--subject", subject},
			} {
				if strings.ContainsAny(f.val, "\r\n") {
					return fmt.Errorf("%s contains illegal CR or LF", f.name)
				}
			}

			// 3. Address parsing.
			toAddrs, err := parseAddressList("--to", to)
			if err != nil {
				return err
			}
			if len(toAddrs) == 0 {
				return fmt.Errorf("--to must contain at least one address")
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

			// 9. Call client.
			client, err := newGmailClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
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
			})
			if err != nil {
				return fmt.Errorf("creating draft: %w", err)
			}

			// 10. Print result.
			if jsonOut {
				return printJSON(result)
			}
			fmt.Printf("Draft created: %s\n", result.ID)
			fmt.Printf("To: %s\n", SanitizeOutput(strings.Join(toAddrs, ", ")))
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

	cmd.Flags().StringVar(&to, "to", "", "Recipient(s), comma-separated")
	cmd.Flags().StringVar(&cc, "cc", "", "Cc recipient(s), comma-separated")
	cmd.Flags().StringVar(&bcc, "bcc", "", "Bcc recipient(s), comma-separated")
	cmd.Flags().StringVar(&fromAddr, "from", "", "From address (Gmail send-as alias)")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Subject line")
	cmd.Flags().StringVar(&body, "body", "", "Body content (markdown by default)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Read body from file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read body from stdin")
	cmd.Flags().BoolVar(&plain, "plain", false, "Send body as plain text (no markdown rendering)")
	cmd.Flags().BoolVar(&htmlMode, "html", false, "Send body as raw HTML (no markdown rendering)")
	cmd.Flags().StringArrayVarP(&attach, "attach", "a", nil, "File path to attach (repeat for multiple)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output result as JSON")

	return cmd
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
	ext := filepath.Ext(path)
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

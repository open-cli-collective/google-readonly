package mail

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-cli-collective/google-readonly/internal/gmail"
	"github.com/open-cli-collective/google-readonly/internal/output"
)

// ClientFactory is the function used to create Gmail clients.
// Override in tests to inject mocks.
var ClientFactory = func() (gmail.GmailClientInterface, error) {
	return gmail.NewClient(context.Background())
}

// newGmailClient creates and returns a new Gmail client
func newGmailClient() (gmail.GmailClientInterface, error) {
	return ClientFactory()
}

// printJSON encodes data as indented JSON to stdout
func printJSON(data any) error {
	return output.JSONStdout(data)
}

// MessagePrintOptions controls which fields to include in message output
type MessagePrintOptions struct {
	IncludeThreadID bool
	IncludeTo       bool
	IncludeSnippet  bool
	IncludeBody     bool
}

// printMessageHeader prints the common header fields of a message
func printMessageHeader(msg *gmail.Message, opts MessagePrintOptions) {
	fmt.Printf("ID: %s\n", msg.ID)
	if opts.IncludeThreadID {
		fmt.Printf("ThreadID: %s\n", msg.ThreadID)
	}
	// Sanitize user-provided content to prevent terminal injection attacks
	fmt.Printf("From: %s\n", SanitizeOutput(msg.From))
	if opts.IncludeTo {
		fmt.Printf("To: %s\n", SanitizeOutput(msg.To))
	}
	fmt.Printf("Subject: %s\n", SanitizeOutput(msg.Subject))
	fmt.Printf("Date: %s\n", msg.Date)
	if len(msg.Labels) > 0 {
		fmt.Printf("Labels: %s\n", strings.Join(msg.Labels, ", "))
	}
	if len(msg.Categories) > 0 {
		fmt.Printf("Categories: %s\n", strings.Join(msg.Categories, ", "))
	}
	if opts.IncludeSnippet {
		fmt.Printf("Snippet: %s\n", SanitizeOutput(msg.Snippet))
	}
	if opts.IncludeBody {
		fmt.Print("\n--- Body ---\n\n")
		fmt.Println(SanitizeOutput(msg.Body))
	}
}

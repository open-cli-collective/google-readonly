package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "preserves newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "preserves tabs",
			input:    "Col1\tCol2\tCol3",
			expected: "Col1\tCol2\tCol3",
		},
		{
			name:     "preserves carriage return",
			input:    "Line1\r\nLine2",
			expected: "Line1\r\nLine2",
		},
		{
			name:     "removes simple color codes",
			input:    "\x1b[31mRed Text\x1b[0m",
			expected: "Red Text",
		},
		{
			name:     "removes bold/reset sequences",
			input:    "\x1b[1mBold\x1b[0m Normal",
			expected: "Bold Normal",
		},
		{
			name:     "removes cursor movement",
			input:    "\x1b[2J\x1b[H\x1b[3AClear and move",
			expected: "Clear and move",
		},
		{
			name:     "removes OSC title sequence",
			input:    "\x1b]0;Evil Title\x07Normal text",
			expected: "Normal text",
		},
		{
			name:     "removes OSC with ST terminator",
			input:    "\x1b]0;Evil Title\x1b\\Normal text",
			expected: "Normal text",
		},
		{
			name:     "removes hyperlink OSC",
			input:    "\x1b]8;;http://evil.com\x07Click me\x1b]8;;\x07",
			expected: "Click me",
		},
		{
			name:     "removes null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "removes bell character",
			input:    "Alert!\x07\x07\x07",
			expected: "Alert!",
		},
		{
			name:     "removes backspace",
			input:    "Overwrite\x08\x08\x08\x08\x08test",
			expected: "Overwritetest",
		},
		{
			name:     "removes form feed",
			input:    "Page1\x0cPage2",
			expected: "Page1Page2",
		},
		{
			name:     "removes vertical tab",
			input:    "Line1\x0bLine2",
			expected: "Line1Line2",
		},
		{
			name:     "removes DEL character",
			input:    "Hello\x7fWorld",
			expected: "HelloWorld",
		},
		{
			name:     "handles multiple escape sequences",
			input:    "\x1b[31m\x1b[1mBold Red\x1b[0m and \x1b[32mGreen\x1b[0m",
			expected: "Bold Red and Green",
		},
		{
			name:     "handles complex CSI sequence",
			input:    "\x1b[38;2;255;0;0mTrue color red\x1b[0m",
			expected: "True color red",
		},
		{
			name:     "preserves escape without valid sequence",
			input:    "Normal \x1b text",
			expected: "Normal \x1b text", // Lone escape without valid sequence is preserved (harmless)
		},
		{
			name:     "preserves unicode text",
			input:    "Hello 世界 🌍",
			expected: "Hello 世界 🌍",
		},
		{
			name:     "preserves email formatting",
			input:    "From: user@example.com\nSubject: Test\n\nBody here",
			expected: "From: user@example.com\nSubject: Test\n\nBody here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename unchanged",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
		},
		{
			name:     "removes ANSI from filename",
			input:    "\x1b[31mevil.exe\x1b[0m",
			expected: "evil.exe",
		},
		{
			name:     "removes RTL override (extension spoofing)",
			input:    "invoice\u202Efdp.exe",
			expected: "invoicefdp.exe",
		},
		{
			name:     "removes LTR override",
			input:    "file\u202Dname.txt",
			expected: "filename.txt",
		},
		{
			name:     "removes multiple bidi characters",
			input:    "\u202Atest\u202B\u202Cfile\u202D.txt\u202E",
			expected: "testfile.txt",
		},
		{
			name:     "removes isolate characters",
			input:    "\u2066\u2067\u2068test\u2069.pdf",
			expected: "test.pdf",
		},
		{
			name:     "preserves unicode in filename",
			input:    "文档.pdf",
			expected: "文档.pdf",
		},
		{
			name:     "removes null byte from filename",
			input:    "file\x00name.txt",
			expected: "filename.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeOutput_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "terminal title attack",
			input:    "\x1b]0;rm -rf /\x07Please review this document",
			expected: "Please review this document",
		},
		{
			name:     "cursor position manipulation",
			input:    "\x1b[1;1H\x1b[2JClearing your screen...",
			expected: "Clearing your screen...",
		},
		{
			name:     "hyperlink with misleading text",
			input:    "\x1b]8;;http://phishing.com\x07https://google.com\x1b]8;;\x07",
			expected: "https://google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

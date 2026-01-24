# CLAUDE.md

This file provides guidance for AI agents working with the gro codebase.

## Project Overview

gro is a **read-only** command-line interface for Google services written in Go. It uses OAuth2 for authentication and only requests read-only scopes - no write, send, or delete operations are possible.

**Binary name:** `gro`
**Module:** `github.com/open-cli-collective/google-readonly`

### Current Features
- Gmail: Search, read, thread viewing, labels, attachments

### Planned Features
- Google Calendar: List calendars, view events
- Google Drive: List files, download content

## Quick Commands

```bash
# Build
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Lint
make lint

# Format code
make fmt

# All checks (format, lint, test)
make verify

# Install locally
make install

# Clean build artifacts
make clean
```

## Architecture

```
google-readonly/
├── main.go                          # Entry point
├── cmd/gro/                         # Main package
│   └── main.go
├── internal/
│   ├── cmd/
│   │   ├── root/                    # Root command, version
│   │   │   └── root.go
│   │   ├── initcmd/                 # OAuth setup (gro init)
│   │   │   ├── init.go
│   │   │   └── init_test.go
│   │   ├── config/                  # gro config {show,test,clear}
│   │   │   ├── config.go
│   │   │   └── config_test.go
│   │   └── mail/                    # gro mail {search,read,thread,labels,attachments}
│   │       ├── mail.go              # Parent command
│   │       ├── search.go
│   │       ├── read.go
│   │       ├── thread.go
│   │       ├── labels.go
│   │       ├── attachments.go
│   │       ├── attachments_list.go
│   │       ├── attachments_download.go
│   │       ├── output.go            # Shared output helpers
│   │       └── *_test.go
│   │
│   ├── gmail/                       # Gmail API client
│   │   ├── client.go
│   │   ├── messages.go
│   │   ├── attachments.go
│   │   └── *_test.go
│   │
│   ├── keychain/                    # Secure credential storage
│   │   ├── keychain.go
│   │   ├── keychain_darwin.go       # macOS Keychain support
│   │   ├── keychain_linux.go        # Linux secret-tool support
│   │   ├── keychain_windows.go      # Windows file fallback
│   │   ├── token_source.go          # Persistent token source wrapper
│   │   └── keychain_test.go
│   │
│   ├── zip/                         # Secure zip extraction
│   │   ├── extract.go
│   │   └── extract_test.go
│   │
│   └── version/                     # Build-time version injection
│       └── version.go
│
├── .github/workflows/
│   ├── ci.yml                       # Lint and test on PR/push
│   ├── auto-release.yml             # Create tags on main push
│   └── release.yml                  # Build and release binaries
│
├── packaging/
│   ├── chocolatey/                  # Windows Chocolatey package
│   └── winget/                      # Windows Winget manifests
│
├── Makefile                         # Build, test, lint targets
├── .goreleaser.yml                  # Cross-platform builds
└── .golangci.yml                    # Linter config (v2 format)
```

## Key Patterns

### Read-Only by Design

This CLI intentionally only supports read operations:
- Uses `gmail.GmailReadonlyScope` exclusively
- Only calls `.List()` and `.Get()` Gmail API methods
- No `.Send()`, `.Delete()`, `.Modify()`, or `.Trash()` operations

### OAuth2 Configuration

Credentials are stored in `~/.config/google-readonly/`:
- `credentials.json` - OAuth client credentials (from Google Cloud Console)

OAuth tokens are stored securely based on platform:
- **macOS**: System Keychain (via `security` CLI)
- **Linux**: libsecret (via `secret-tool`) if available, otherwise config file
- **Fallback**: `~/.config/google-readonly/token.json` with 0600 permissions

### Command Patterns

All commands use the factory pattern with `NewCommand()`:

```go
func NewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "search <query>",
        Short: "Search for messages",
        Args:  cobra.ExactArgs(1),
        RunE:  runSearch,
    }
    cmd.Flags().Int64VarP(&searchMaxResults, "max", "m", 10, "Maximum results")
    return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
    client, err := newGmailClient()
    if err != nil {
        return err
    }
    // ... use client
}
```

### Output Formats

Commands support two output modes:
- **Text** (default): Human-readable formatted output
- **JSON** (`--json`): Machine-readable JSON for scripting

```go
if jsonOutput {
    return printJSON(messages)
}
// ... text output
```

## Testing

Tests use `testify` for assertions and table-driven test patterns:

```go
func TestParseMessage(t *testing.T) {
    tests := []struct {
        name     string
        input    *gmail.Message
        expected *Message
    }{
        {"basic message", ...},
        {"multipart message", ...},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseMessage(tt.input, true)
            assert.Equal(t, tt.expected.Subject, result.Subject)
        })
    }
}
```

Run tests: `make test`

Coverage report: `make test-cover && open coverage.html`

## Adding a New Command

1. Create new file in appropriate `internal/cmd/` directory
2. Define the command with `NewCommand()` factory function
3. Register in parent command's `NewCommand()` with `AddCommand()`
4. Add flags if needed
5. Write tests in `*_test.go`

Example:

```go
func NewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "labels",
        Short: "List Gmail labels",
        RunE:  runLabels,
    }
    cmd.Flags().BoolVarP(&labelsJSON, "json", "j", false, "Output as JSON")
    return cmd
}
```

## Dependencies

Key dependencies:
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/oauth2` - OAuth2 client
- `google.golang.org/api/gmail/v1` - Gmail API client
- `github.com/stretchr/testify` - Testing assertions (dev)

## Error Message Conventions

Follow [Go Code Review Comments](https://github.com/go/wiki/wiki/CodeReviewComments#error-strings):

- Start with lowercase
- Don't end with punctuation
- Be descriptive but concise

```go
// Good
return fmt.Errorf("failed to get message: %w", err)
return fmt.Errorf("attachment not found: %s", filename)

// Bad
return fmt.Errorf("Failed to get message: %w", err)  // capitalized
return fmt.Errorf("attachment not found.")           // ends with punctuation
```

## Commit Conventions

Use conventional commits:

```
type(scope): description

feat(mail): add attachment download command
fix(keychain): handle missing secret-tool
docs(readme): add installation instructions
```

| Prefix | Purpose | Triggers Release? |
|--------|---------|-------------------|
| `feat:` | New features | Yes |
| `fix:` | Bug fixes | Yes |
| `docs:` | Documentation only | No |
| `test:` | Adding/updating tests | No |
| `refactor:` | Code changes that don't fix bugs or add features | No |
| `chore:` | Maintenance tasks | No |
| `ci:` | CI/CD changes | No |

## CI & Release Workflow

Releases are automated with a dual-gate system:

**Gate 1 - Path filter:** Only triggers when Go code changes (`**.go`, `go.mod`, `go.sum`)
**Gate 2 - Commit prefix:** Only `feat:` and `fix:` commits create releases

This means:
- `feat: add command` + Go files changed → release
- `fix: handle edge case` + Go files changed → release
- `docs:`, `ci:`, `test:`, `refactor:` → no release
- Changes only to docs, packaging, workflows → no release

## Common Issues

### "Unable to read credentials file"

Ensure OAuth credentials are set up:
```bash
mkdir -p ~/.config/google-readonly
# Download credentials.json from Google Cloud Console
mv ~/Downloads/client_secret_*.json ~/.config/google-readonly/credentials.json
```

### "Token has been expired or revoked"

Clear the token and re-authenticate:
```bash
gro config clear
gro init
```

## Security

- **Read-only scope**: Cannot modify, send, or delete data
- **Secure token storage**: OAuth tokens stored in system keychain when available
- **File fallback**: When secure storage is unavailable, tokens stored with 0600 permissions
- **Token refresh persistence**: Refreshed tokens are automatically saved
- **No credential exposure**: Credentials never logged or transmitted

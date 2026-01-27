# google-readonly

A read-only command-line interface for Google services. Search, read, and view Gmail messages, threads, and attachments without any ability to modify, send, or delete data.

## Features

- **Read-only access** - Uses `gmail.readonly`, `calendar.readonly`, `contacts.readonly`, and `drive.readonly` OAuth scopes
- **Gmail support** - Search messages, read content, view threads, list labels, download attachments
- **Calendar support** - List calendars, view events, today/week shortcuts
- **Contacts support** - List contacts, search, view details, list groups
- **Drive support** - List files, search, view metadata, download files, folder tree
- **JSON output** - Machine-readable output for scripting
- **Secure storage** - OAuth tokens stored in system keychain (macOS/Linux)

## Installation

### macOS

**Homebrew (recommended)**

```bash
brew install open-cli-collective/tap/google-readonly
```

> Note: This installs from our third-party tap.

---

### Windows

**Chocolatey**

```powershell
choco install google-readonly
```

**Winget**

```powershell
winget install OpenCLICollective.google-readonly
```

---

### Linux

**APT (Debian/Ubuntu)**

```bash
# Add the GPG key
curl -fsSL https://open-cli-collective.github.io/linux-packages/keys/gpg.asc | sudo gpg --dearmor -o /usr/share/keyrings/open-cli-collective.gpg

# Add the repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/open-cli-collective.gpg] https://open-cli-collective.github.io/linux-packages/apt stable main" | sudo tee /etc/apt/sources.list.d/open-cli-collective.list

# Install
sudo apt update
sudo apt install google-readonly
```

> Note: This is our third-party APT repository, not official Debian/Ubuntu repos.

**DNF/YUM (Fedora/RHEL/CentOS)**

```bash
# Add the repository
sudo tee /etc/yum.repos.d/open-cli-collective.repo << 'EOF'
[open-cli-collective]
name=Open CLI Collective
baseurl=https://open-cli-collective.github.io/linux-packages/rpm
enabled=1
gpgcheck=1
gpgkey=https://open-cli-collective.github.io/linux-packages/keys/gpg.asc
EOF

# Install
sudo dnf install google-readonly
```

> Note: This is our third-party RPM repository, not official Fedora/RHEL repos.

**Binary download**

Download `.deb`, `.rpm`, or `.tar.gz` from the [Releases page](https://github.com/open-cli-collective/google-readonly/releases).

---

### From Source

```bash
go install github.com/open-cli-collective/google-readonly/cmd/gro@latest
```

## Setup

### 1. Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the required APIs:
   - Go to **APIs & Services** > **Library**
   - Search for and enable: **Gmail API**
   - Enable: **Google Calendar API**
   - Enable: **People API** (for Contacts)
   - Enable: **Google Drive API**

### 2. Create OAuth Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. If prompted, configure the OAuth consent screen:
   - Choose **External** user type
   - Fill in required fields (app name, support email)
   - Add scopes:
     - `https://www.googleapis.com/auth/gmail.readonly`
     - `https://www.googleapis.com/auth/calendar.readonly`
     - `https://www.googleapis.com/auth/contacts.readonly`
     - `https://www.googleapis.com/auth/drive.readonly`
   - Add your email as a test user
4. For Application type, select **Desktop app**
5. Click **Create**
6. Download the JSON file

### 3. Configure gro

1. Create the config directory:
   ```bash
   mkdir -p ~/.config/google-readonly
   ```

2. Move the downloaded credentials file:
   ```bash
   mv ~/Downloads/client_secret_*.json ~/.config/google-readonly/credentials.json
   ```

### 4. Authenticate

Run the init command to complete OAuth setup:

```bash
gro init
```

1. A URL will be displayed - open it in your browser
2. Sign in with your Google account
3. Grant read-only access
4. Your browser will redirect to a localhost URL (the error page is expected)
5. Copy the entire URL or just the authorization code
6. Paste it back into the terminal

Your token will be saved securely (system keychain on macOS/Linux, or `~/.config/google-readonly/token.json` as fallback).

## Commands

### Configuration Commands

```bash
# Guided OAuth setup
gro init

# Check configuration status
gro config show

# Test API connectivity
gro config test

# Clear stored OAuth token
gro config clear

# View cache status
gro config cache show

# Clear cached data
gro config cache clear

# Set cache TTL (in hours)
gro config cache ttl 12

# Show version
gro --version
```

### Gmail Commands

All Gmail commands are under `gro mail`:

```bash
# Search messages
gro mail search "is:unread"
gro mail search "from:someone@example.com" --max 20
gro mail search "subject:meeting" --json

# Read a message
gro mail read <message-id>
gro mail read <message-id> --json

# View conversation thread
gro mail thread <thread-id>
gro mail thread <message-id> --json

# List labels
gro mail labels
gro mail labels --json

# List attachments
gro mail attachments list <message-id>
gro mail attachments list <message-id> --json

# Download attachments
gro mail attachments download <message-id> --all
gro mail attachments download <message-id> --filename report.pdf
gro mail attachments download <message-id> --all --output ~/Downloads
gro mail attachments download <message-id> --filename archive.zip --extract
```

### Calendar Commands

All Calendar commands are under `gro calendar` (or `gro cal`):

```bash
# List all calendars
gro calendar list
gro cal list --json

# List upcoming events
gro calendar events
gro cal events --max 20
gro cal events --from 2026-01-01 --to 2026-01-31

# Get event details
gro calendar get <event-id>
gro cal get <event-id> --json

# Today's events
gro calendar today
gro cal today --json

# This week's events
gro calendar week
gro cal week --json
```

## Command Reference

### gro init

Guided setup for Google API OAuth authentication.

```
Usage: gro init [flags]

Flags:
      --no-verify   Skip connectivity verification after setup
```

### gro config show

Display current configuration status including credentials and token.

```
Usage: gro config show
```

### gro config test

Test Gmail API connectivity with current credentials.

```
Usage: gro config test
```

### gro config clear

Remove stored OAuth token (forces re-authentication).

```
Usage: gro config clear
```

### gro config cache show

Display cache status including location, TTL, and cached data status.

```
Usage: gro config cache show [flags]

Flags:
  -j, --json       Output as JSON
```

### gro config cache clear

Remove all cached data. Cache will be repopulated on next use.

```
Usage: gro config cache clear
```

### gro config cache ttl

Set the cache time-to-live in hours.

```
Usage: gro config cache ttl <hours>
```

### gro mail search

Search for Gmail messages using Gmail's search syntax.

```
Usage: gro mail search <query> [flags]

Flags:
  -m, --max int    Maximum number of results (default 10)
  -j, --json       Output as JSON
```

### gro mail read

Read the full content of a Gmail message by its ID.

```
Usage: gro mail read <message-id> [flags]

Flags:
  -j, --json       Output as JSON
```

### gro mail thread

Read all messages in a Gmail conversation thread.

```
Usage: gro mail thread <id> [flags]

Flags:
  -j, --json       Output as JSON
```

### gro mail labels

List all Gmail labels including user labels and system categories.

```
Usage: gro mail labels [flags]

Flags:
  -j, --json       Output as JSON
```

### gro mail attachments list

List all attachments in a Gmail message.

```
Usage: gro mail attachments list <message-id> [flags]

Flags:
  -j, --json       Output as JSON
```

### gro mail attachments download

Download attachments from a Gmail message.

```
Usage: gro mail attachments download <message-id> [flags]

Flags:
  -f, --filename string   Download only this attachment
  -o, --output string     Output directory (default ".")
  -a, --all               Download all attachments
  -e, --extract           Extract zip files after download
```

### gro calendar list

List all calendars the user has access to.

```
Usage: gro calendar list [flags]

Aliases: gro cal list

Flags:
  -j, --json       Output as JSON
```

### gro calendar events

List events from a calendar.

```
Usage: gro calendar events [calendar-id] [flags]

Aliases: gro cal events

Flags:
  -c, --calendar string   Calendar ID to query (default "primary")
  -m, --max int           Maximum number of events (default 10)
      --from string       Start date (YYYY-MM-DD)
      --to string         End date (YYYY-MM-DD)
  -j, --json              Output as JSON
```

### gro calendar get

Get the full details of a calendar event.

```
Usage: gro calendar get <event-id> [flags]

Aliases: gro cal get

Flags:
  -c, --calendar string   Calendar ID containing the event (default "primary")
  -j, --json              Output as JSON
```

### gro calendar today

Show all events for today.

```
Usage: gro calendar today [flags]

Aliases: gro cal today

Flags:
  -c, --calendar string   Calendar ID to query (default "primary")
  -j, --json              Output as JSON
```

### gro calendar week

Show all events for the current week (Monday to Sunday).

```
Usage: gro calendar week [flags]

Aliases: gro cal week

Flags:
  -c, --calendar string   Calendar ID to query (default "primary")
  -j, --json              Output as JSON
```

### Contacts Commands

All Contacts commands are under `gro contacts` (or `gro ppl`):

```bash
# List all contacts
gro contacts list
gro ppl list --max 20
gro contacts list --json

# Search contacts
gro contacts search "John"
gro ppl search "example.com" --max 20

# Get contact details
gro contacts get people/c123456789
gro ppl get people/c123456789 --json

# List contact groups
gro contacts groups
gro ppl groups --json
```

### gro contacts list

List all contacts sorted by last name.

```
Usage: gro contacts list [flags]

Aliases: gro ppl list

Flags:
  -m, --max int    Maximum number of contacts (default 10)
  -j, --json       Output as JSON
```

### gro contacts search

Search contacts by name, email, phone, or organization.

```
Usage: gro contacts search <query> [flags]

Aliases: gro ppl search

Flags:
  -m, --max int    Maximum number of results (default 10)
  -j, --json       Output as JSON
```

### gro contacts get

Get the full details of a specific contact.

```
Usage: gro contacts get <resource-name> [flags]

Aliases: gro ppl get

Flags:
  -j, --json       Output as JSON
```

### gro contacts groups

List all contact groups (labels).

```
Usage: gro contacts groups [flags]

Aliases: gro ppl groups

Flags:
  -m, --max int    Maximum number of groups (default 30)
  -j, --json       Output as JSON
```

### Drive Commands

All Drive commands are under `gro drive` (or `gro files`):

```bash
# List files in root or folder
gro drive list
gro files list --max 20
gro drive list <folder-id> --type document

# Search files
gro drive search "quarterly report"
gro files search --name "budget" --type spreadsheet
gro drive search --modified-after 2024-01-01

# Get file metadata
gro drive get <file-id>
gro files get <file-id> --json

# Download files
gro drive download <file-id>
gro files download <file-id> --output ./report.pdf
gro drive download <file-id> --format pdf  # Export Google Doc as PDF
gro drive download <file-id> --stdout       # Write to stdout

# Show folder tree
gro drive tree
gro files tree <folder-id> --depth 3
gro drive tree --files  # Include files, not just folders
```

### gro drive list

List files in Google Drive root or a specific folder.

```
Usage: gro drive list [folder-id] [flags]

Aliases: gro files list

Flags:
  -m, --max int      Maximum number of files (default 25)
  -t, --type string  Filter by type (document, spreadsheet, presentation, folder, pdf, image, video, audio)
  -j, --json         Output as JSON
```

### gro drive search

Search for files in Google Drive.

```
Usage: gro drive search [query] [flags]

Aliases: gro files search

Flags:
  -n, --name string            Search by filename only
  -t, --type string            Filter by file type
      --owner string           Filter by owner (me, or email)
      --modified-after string  Modified after date (YYYY-MM-DD)
      --modified-before string Modified before date (YYYY-MM-DD)
      --in-folder string       Search within folder ID
  -m, --max int                Maximum results (default 25)
  -j, --json                   Output as JSON
```

### gro drive get

Get detailed metadata for a file.

```
Usage: gro drive get <file-id> [flags]

Aliases: gro files get

Flags:
  -j, --json       Output as JSON
```

### gro drive download

Download a file or export a Google Workspace file.

```
Usage: gro drive download <file-id> [flags]

Aliases: gro files download

Flags:
  -o, --output string   Output file path
  -f, --format string   Export format for Google Workspace files
      --stdout          Write to stdout instead of file
```

Export formats for Google Workspace files:
- **Documents**: pdf, docx, txt, html, md, rtf, odt
- **Spreadsheets**: pdf, xlsx, csv, tsv, ods
- **Presentations**: pdf, pptx, odp
- **Drawings**: pdf, png, svg, jpg

### gro drive tree

Display folder structure as a tree.

```
Usage: gro drive tree [folder-id] [flags]

Aliases: gro files tree

Flags:
  -d, --depth int    Maximum depth to traverse (default 2)
      --files        Include files in addition to folders
  -j, --json         Output as JSON
```

## Search Query Reference

gro supports all Gmail search operators:

| Operator | Example | Description |
|----------|---------|-------------|
| `from:` | `from:alice@example.com` | Messages from sender |
| `to:` | `to:bob@example.com` | Messages to recipient |
| `subject:` | `subject:meeting` | Subject contains word |
| `is:` | `is:unread`, `is:starred` | Message state |
| `has:` | `has:attachment` | Has attachment |
| `filename:` | `filename:pdf`, `filename:xlsx` | Attachment file type |
| `after:` | `after:2024/01/01` | After date |
| `before:` | `before:2024/02/01` | Before date |
| `label:` | `label:work` | Has label |
| `category:` | `category:updates` | In category |
| `in:` | `in:inbox`, `in:sent` | In folder |
| `larger:` | `larger:5M` | Larger than size |
| `smaller:` | `smaller:1M` | Smaller than size |

See [Gmail search operators](https://support.google.com/mail/answer/7190) for the complete list.

## Shell Completion

gro supports tab completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Load in current session
source <(gro completion bash)

# Install permanently (Linux)
gro completion bash | sudo tee /etc/bash_completion.d/gro > /dev/null

# Install permanently (macOS with Homebrew)
gro completion bash > $(brew --prefix)/etc/bash_completion.d/gro
```

### Zsh

```bash
# Load in current session
source <(gro completion zsh)

# Install permanently
mkdir -p ~/.zsh/completions
gro completion zsh > ~/.zsh/completions/_gro

# Add to ~/.zshrc if not already present:
# fpath=(~/.zsh/completions $fpath)
# autoload -Uz compinit && compinit
```

### Fish

```bash
# Load in current session
gro completion fish | source

# Install permanently
gro completion fish > ~/.config/fish/completions/gro.fish
```

### PowerShell

```powershell
# Load in current session
gro completion powershell | Out-String | Invoke-Expression

# Install permanently (add to $PROFILE)
gro completion powershell >> $PROFILE
```

## Configuration

Configuration files are stored in `~/.config/google-readonly/`:

| File | Description |
|------|-------------|
| `credentials.json` | OAuth client credentials (from Google Cloud Console) |
| `token.json` | OAuth access/refresh token (fallback if keychain unavailable) |
| `config.json` | User settings (cache TTL, etc.) |
| `cache/` | Cached API metadata for faster repeated lookups |

### Cache Settings

gro caches Drive metadata (like shared drive lists) to speed up repeated commands. The cache TTL is configured during `gro init` (default: 24 hours).

```bash
# View cache status
gro config cache show

# Clear cache
gro config cache clear

# Change cache TTL
gro config cache ttl 12    # Set to 12 hours
```

The cache is automatically repopulated when stale or after being cleared.

## Security

- This tool only requests **read-only** access to Google services
- No write, send, or delete operations are possible
- OAuth tokens are stored in system keychain (macOS Keychain / Linux secret-tool) when available
- File-based storage uses `0600` permissions
- Credentials never leave your machine
- Zip extraction includes security safeguards (size limits, path traversal prevention)

## Troubleshooting

### "Unable to read credentials file"

Ensure `credentials.json` exists:
```bash
ls -la ~/.config/google-readonly/credentials.json
```

### "Token has been expired or revoked"

Clear the token and re-authenticate:
```bash
gro config clear
gro init
```

### "Access blocked: This app's request is invalid"

Your OAuth consent screen may not be properly configured. Ensure:
1. All required APIs are enabled (Gmail, Calendar, People, Drive)
2. Your email is added as a test user (for apps in testing mode)
3. The required scopes are added

### "API has not been used in project" or "SERVICE_DISABLED"

The specific Google API hasn't been enabled in your Cloud project:
1. Check the error message for the activation URL
2. Visit the URL and click **Enable**
3. Wait a few minutes for propagation
4. Clear your token and re-authenticate:
   ```bash
   gro config clear
   gro init
   ```

### "Request had invalid authentication credentials"

Your token may be missing scopes for a newly added service. Clear and re-authenticate:
```bash
gro config clear
gro init
```

## License

MIT License - see [LICENSE](LICENSE) for details.

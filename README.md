# google-readonly

A non-destructive command-line interface for Google services. Search, read, and organize Gmail messages, calendar events, contacts, and Drive files. Supports labeling, archiving, starring, RSVP, and group management. No send, delete, or trash operations are possible.

## Features

- **Non-destructive by design** - Read access plus organizational operations; no send, delete, or trash
- **Gmail support** - Search messages, read content, view threads, list labels, download attachments, archive, star, label, categorize, mark read/unread, compose drafts (never sent automatically)
- **Calendar support** - List calendars, view events, today/week shortcuts, RSVP, color-coding
- **Contacts support** - List contacts, search, view details, list groups, star, group management
- **Drive support** - List files, search, view metadata, download files, folder tree, star/unstar
- **Bulk operations** - Pipe IDs between commands, use search queries inline, or pass IDs as arguments
- **Text-first output** - Resource-surface commands emit token-dense text only. JSON is reserved for control-plane envelopes (`gro refresh --json`, `gro config show --json`); see cli-common `docs/output-and-rendering.md` §2. **Breaking change in #144:** per-command `--json` on resource reads/mutations has been removed.
- **Secure storage** - the OAuth token is stored only in the OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager, or an opt-in encrypted file) via the shared `cli-common/credstore`
- **Single-run guided setup** - `gro init` reads the OAuth client JSON from clipboard / paste / file path (your admin may share one via 1Password) and walks you through OAuth in one shot; `gro me` confirms identity afterwards

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

> **Are you a Google Workspace admin setting up `gro` for your whole org?** See [WORKSPACE_ADMINS.md](./WORKSPACE_ADMINS.md) — set up an Internal OAuth app once, distribute `credentials.json` to users via 1Password, and skip Google's app verification process. Otherwise, follow the DIY setup below.

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
     - `https://www.googleapis.com/auth/gmail.modify` (read + archive/label/star)
     - `https://www.googleapis.com/auth/calendar.readonly` (read calendar data)
     - `https://www.googleapis.com/auth/calendar.events` (RSVP, color-coding)
     - `https://www.googleapis.com/auth/contacts` (read + star/group management)
     - `https://www.googleapis.com/auth/drive.readonly` (read Drive files)
     - `https://www.googleapis.com/auth/drive.metadata` (star/unstar files)
   - Add your email as a test user
4. For Application type, select **Desktop app**
5. Click **Create**
6. Download the JSON file

> **Note on scopes:** gro uses the `gmail.modify` scope (a superset of `gmail.readonly`) because organizational operations like archive, label, and star require it. Similarly, `contacts` and `calendar.events` scopes enable starring, group management, and RSVP. No send, delete, or trash operations are possible regardless of scope.

### 3. Publish Your OAuth App (Recommended)

By default, new OAuth apps are in **"Testing"** mode, which causes **tokens to expire after 7 days**. To avoid frequent re-authentication:

1. Go to **Google Auth Platform** > **Audience** (or **OAuth consent screen** in older UI)
2. Find the **Publishing status** section
3. Click **Publish app**

For personal use with these scopes, publishing is straightforward and doesn't require Google verification. Once published, tokens will last until revoked or unused for 6 months.

> **Note:** If you skip this step, you'll need to run `gro init` every 7 days to re-authenticate.

### 4. Run the wizard

After creating the OAuth credentials in step 2 (or 3), run:

```bash
gro init
```

The wizard will:

1. **Ingest credentials.json.** Pick one of three options when prompted:
   - **Read from clipboard** — copy the JSON in the Cloud Console and choose this; gro reads, validates, and saves it.
   - **Paste in terminal** — paste the JSON directly into the terminal.
   - **Point to a file path** — type the path to the downloaded JSON.

2. **Open the consent URL.** Confirm to auto-open your browser, or copy the URL manually.

3. **Sign in and paste the redirect URL back.** After clicking "Allow", your browser redirects to a localhost URL that shows an error — that's expected. Copy the entire URL (or just the `code=` value) and paste it into the wizard.

4. **Set the cache TTL** (first-run only). The wizard asks how many hours to cache Drive metadata. Press Enter to accept the default (24h).

The token is saved only in the OS keyring via `cli-common/credstore` (macOS Keychain, Linux Secret Service, Windows Credential Manager, or an opt-in encrypted file). Backend selection has three user-configurable knobs that fall back to auto-detect, in precedence order: `--backend <name>` flag > `GOOGLE_READONLY_KEYRING_BACKEND` env var > `keyring.backend` in `config.yml` > auto-detect. Supported names: `keychain`, `wincred`, `secret-service`, `file`, `memory`. The `file` backend additionally requires `GOOGLE_READONLY_KEYRING_PASSPHRASE`. There is no plaintext `token.json` fallback.

When init succeeds, it prints the same `gro me` one-liner as its proof-of-life. You can re-run `gro me` any time after.

**Linux clipboard prerequisites.** The clipboard option requires `xclip` or `xsel` to be installed. If neither is available, the menu falls back to manual paste / file path automatically.

**Useful flags:**

```bash
gro init --credentials-file ~/Downloads/client_secret.json   # bypass the wizard
gro init --no-browser                                        # don't auto-open
gro init --no-verify                                         # skip post-setup API check
```

### Non-interactive ingress (CI / automation)

For unattended installs you can seed the token without the browser flow using
`gro set-credential`. It accepts a serialized `oauth2.Token` JSON object
**only** via stdin or a named environment variable — never as a flag or
positional argument — and only for the key `oauth_token`. The value is never
echoed.

```bash
# From a secrets manager via stdin
op read 'op://Vault/gro/oauth_token' | gro set-credential --key oauth_token --stdin

# From an environment variable
gro set-credential --key oauth_token --from-env GRO_OAUTH_TOKEN

# Target a non-default credential ref (default: config.yml credential_ref)
gro set-credential --key oauth_token --stdin --ref my-service/profile

# Two-phase install: feed the OAuth redirect code on stdin instead of pasting it
gro init --auth-code-stdin < auth_code.txt
```

Flags:

| Flag | Purpose |
|------|---------|
| `--key oauth_token` | Required. Only `oauth_token` is accepted. |
| `--stdin` | Read the token JSON from stdin. |
| `--from-env NAME` | Read the token JSON from the named environment variable. |
| `--ref SERVICE/PROFILE` | Target credential ref. Defaults to `config.yml`'s `credential_ref`. |

The token lands in whichever backend the standard precedence resolves
(`--backend` flag > `GOOGLE_READONLY_KEYRING_BACKEND` > `keyring.backend`
config > auto). With the default ref, `set-credential`
runs the one-time legacy migration first so a pre-existing `token.json` cannot
later collide; an explicit `--ref` never migrates.

## Commands

### Configuration Commands

```bash
# Guided OAuth setup
gro init

# Show currently authenticated user (resourceName | displayName | primaryEmail)
gro me

# Just the primary email (scriptable)
gro me --id

# Adds granted scopes, token expiry, and storage backend
gro me --extended


# Check configuration status
gro config show

# Test API connectivity
gro config test

# Clear stored OAuth token
gro config clear

# Show version
gro --version

# Enable verbose output for debugging (available on all commands)
gro --verbose <command>
gro -v <command>
```

### Gmail Commands

All Gmail commands are under `gro mail`:

```bash
# Search messages
gro mail search "is:unread"
gro mail search "from:someone@example.com" --max 20
gro mail search "is:starred" --ids          # Output IDs only (for piping)

# Read a message
gro mail read <message-id>

# View conversation thread
gro mail thread <thread-id>

# List labels
gro mail labels

# List attachments
gro mail attachments list <message-id>

# Download attachments
gro mail attachments download <message-id> --all
gro mail attachments download <message-id> --filename report.pdf
gro mail attachments download <message-id> --all --output ~/Downloads
gro mail attachments download <message-id> --filename archive.zip --extract

# Archive messages (remove from inbox)
gro mail archive <id1> <id2>
gro mail archive --query "from:noreply older_than:30d"
gro mail search "from:noreply" --ids | gro mail archive --stdin

# Star / unstar messages
gro mail star <id1> <id2>
gro mail unstar <id>

# Mark read / unread
gro mail mark-read <id1> <id2>
gro mail mark-unread <id>

# Add / remove labels
gro mail label "Work" <id1> <id2>
gro mail unlabel "Promotions" <id>

# Recategorize messages
gro mail categorize promotions <id1> <id2>

# Compose a draft (markdown body by default, rendered to HTML)
gro mail draft --to alice@example.com --subject "Hi" --body "**hello**"
gro mail draft --to "a@x.com, b@x.com" --subject "Sync" --file notes.md
echo "# Hello" | gro mail draft --to a@x.com --subject "Hi" --stdin

# Plain text or raw HTML body
gro mail draft --to a@x.com --subject "Plain" --body "no formatting" --plain
gro mail draft --to a@x.com --subject "Raw" --body "<h1>Hi</h1>" --html

# With attachments
gro mail draft --to a@x.com --subject "See attached" --body "..." --attach report.pdf

# From a send-as alias
gro mail draft --from work@me.com --to a@x.com --subject "Hi" --body "..."

# Reply to an existing message (preserves thread, adds In-Reply-To / References)
gro mail draft --reply-to <message-id> --body "thanks, will review"
gro mail draft --reply-to <message-id>                       # quote-only reply (no typed text)
gro mail draft --reply-to <message-id> --no-quote --body "ack"
gro mail draft --reply-to <message-id> --reply-all --body "looping everyone in"
gro mail draft --reply-to <message-id> --subject "Re: customised" --body "..."
```

Drafts always land in your Gmail Drafts folder for human review. The CLI never calls `drafts.send` — sending requires explicit action in Gmail.

With `--reply-to`, `--to` and `--subject` are derived from the source message (To = original From; Subject = `Re: <original>`, no double prefix). Explicit `--to` / `--cc` / `--subject` flags override the derived values. `--reply-all` adds the original To and Cc as Cc on the reply, filtered to remove your own address (and any `--from` alias).

By default a reply quotes the source message below your text, like Gmail's web UI: an `On <date> <sender> wrote:` line followed by the original body (`> ` line prefixes for plain replies; a collapsible `gmail_quote` block for HTML/markdown replies, which Gmail's UI hides behind its “…” toggle). A body source is optional when replying — a bare `--reply-to` yields a quote-only draft. Use `--no-quote` to reply without quoting (with no body source the draft is left blank). Source bodies are quoted the way Gmail itself does on reply: a plain-text source is escaped and shown with `> ` / `<br>`, while an HTML source (common for alert/marketing/SaaS senders that ship no plain-text part) is nested as HTML so it renders normally rather than appearing as escaped tags. The attribution line is always HTML-escaped.

### Calendar Commands

All Calendar commands are under `gro calendar` (or `gro cal`):

```bash
# List all calendars
gro calendar list

# List upcoming events
gro calendar events
gro cal events --max 20
gro cal events --from 2026-01-01 --to 2026-01-31

# Get event details
gro calendar get <event-id>

# Today's events
gro calendar today

# This week's events
gro calendar week

# RSVP to an event
gro calendar rsvp <event-id> accept
gro cal rsvp <event-id> decline
gro cal rsvp <event-id> tentative

# Set event color
gro calendar color <event-id> tomato
gro cal color <event-id> lavender
```

### Contacts Commands

All Contacts commands are under `gro contacts` (or `gro ppl`):

```bash
# List all contacts
gro contacts list
gro ppl list --max 20
gro contacts list --ids                     # Output resource names only

# Search contacts
gro contacts search "John"
gro ppl search "example.com" --max 20
gro contacts search "John" --ids            # Output resource names only

# Get contact details
gro contacts get people/c123456789

# List contact groups
gro contacts groups

# Star / unstar contacts
gro contacts star people/c123 people/c456
gro contacts unstar people/c123
gro contacts search "John" --ids | gro contacts star --stdin

# Add / remove from groups
gro contacts add-to-group "Friends" people/c123 people/c456
gro contacts remove-from-group "Friends" people/c123
gro contacts add-to-group "VIP" --query "John"
```

### Drive Commands

All Drive commands are under `gro drive` (or `gro files`):

```bash
# List files in root or folder
gro drive list
gro files list --max 20
gro drive list <folder-id> --type document
gro drive list --ids                        # Output file IDs only

# Search files
gro drive search "quarterly report"
gro files search "budget" --name --type spreadsheet
gro drive search --modified-after 2024-01-01
gro drive search "budget" --ids             # Output file IDs only

# Get file metadata
gro drive get <file-id>

# Download files
gro drive download <file-id>
gro files download <file-id> --output ./report.pdf
gro drive download <file-id> --format pdf  # Export Google Doc as PDF
gro drive download <file-id> --stdout       # Write to stdout

# Show folder tree
gro drive tree
gro files tree <folder-id> --depth 3
gro drive tree --files  # Include files, not just folders

# Star / unstar files
gro drive star <file-id>
gro drive unstar <file-id>
gro drive search "budget" --ids | gro drive star --stdin
```

#### Shared Drives

gro supports Google Shared Drives (formerly Team Drives). By default, search includes files from all drives you have access to.

```bash
# List available shared drives
gro drive drives

# Search all drives (default)
gro drive search "quarterly report"

# Search only your personal drive
gro drive search "quarterly report" --my-drive

# Search a specific shared drive by name
gro drive search "budget" --drive "Finance Team"
gro drive list --drive "Engineering"
gro drive tree --drive "Marketing"
```

The `--my-drive` and `--drive` flags are mutually exclusive. Shared drive names are cached locally for fast lookups. Run `gro refresh drives` to refresh the cache, or `gro refresh --status` to inspect freshness.

### Bulk Operations

All organizational commands (archive, star, label, etc.) accept IDs through three input modes:

```bash
# 1. Positional arguments
gro mail archive id1 id2 id3

# 2. Stdin (pipe from search/list with --ids)
gro mail search "from:noreply older_than:30d" --ids | gro mail archive --stdin
gro contacts search "John" --ids | gro contacts star --stdin
gro drive search "budget" --ids | gro drive star --stdin

# 3. Inline query (resolved automatically)
gro mail archive --query "from:noreply older_than:30d"
gro contacts add-to-group "VIP" --query "John"
gro drive star --query "budget"
```

All organizational commands also support `--dry-run` / `-n` to preview changes without applying them.

## Command Reference

### gro init

Guided setup for Google API OAuth authentication.

```
Usage: gro init [flags]

Flags:
      --auth-code-stdin           Read the OAuth authorization code/redirect URL from stdin (two-phase install; implies no browser-open)
      --credentials-file string   Path to a downloaded OAuth client JSON (bypasses the wizard)
      --no-browser                Don't try to open the consent URL in a browser
      --no-verify                 Skip connectivity verification after setup
```

### gro me

Show the currently authenticated Google account.

```
Usage: gro me [flags]

Flags:
      --id         Print only the primary email (scriptable)
      --extended   Add granted scopes, token expiry, and storage backend
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

Remove the stored OAuth token (forces re-authentication). `--all` also removes
`config.yml` and the Drive metadata cache; `--dry-run` reports what would be
removed without removing anything.

```
Usage: gro config clear [--all] [--dry-run]
```

### gro mail search

Search for Gmail messages using Gmail's search syntax.

```
Usage: gro mail search <query> [flags]

Flags:
  -m, --max int    Maximum number of results (default 10)
      --ids        Output only message IDs (one per line, for piping)
```


### gro mail read

Read the full content of a Gmail message by its ID.

```
Usage: gro mail read <message-id> [flags]

Flags:
```

### gro mail thread

Read all messages in a Gmail conversation thread.

```
Usage: gro mail thread <id> [flags]

Flags:
```

### gro mail labels

List all Gmail labels including user labels and system categories.

```
Usage: gro mail labels [flags]

Flags:
```

### gro mail attachments list

List all attachments in a Gmail message.

```
Usage: gro mail attachments list <message-id> [flags]

Flags:
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

### gro mail archive

Archive messages (remove from inbox).

```
Usage: gro mail archive [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail star

Star messages.

```
Usage: gro mail star [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail unstar

Unstar messages.

```
Usage: gro mail unstar [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail mark-read

Mark messages as read.

```
Usage: gro mail mark-read [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail mark-unread

Mark messages as unread.

```
Usage: gro mail mark-unread [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail label

Add a label to messages.

```
Usage: gro mail label <label-name> [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail unlabel

Remove a label from messages.

```
Usage: gro mail unlabel <label-name> [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail categorize

Recategorize messages. Valid categories: personal, social, promotions, updates, forums.

```
Usage: gro mail categorize <category> [message-ids...] [flags]

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read message IDs from stdin
      --query string   Search query to resolve message IDs
```

### gro mail draft

Compose a Gmail draft and save it to the Drafts folder. The CLI never calls `drafts.send`; the draft sits in Gmail for human review and explicit send.

Body input is markdown by default and rendered to HTML. Use `--plain` for plain text or `--html` for raw HTML (no rendering). Body source is one of `--body`, `--stdin`, or `--file`.

```
Usage: gro mail draft [flags]

Flags:
      --to string         Recipient(s), comma-separated (required)
      --cc string         Cc recipient(s), comma-separated
      --bcc string        Bcc recipient(s), comma-separated
      --from string       From address (Gmail send-as alias)
  -s, --subject string    Subject line (required; empty string allowed)
      --body string       Body content (markdown by default)
  -f, --file string       Read body from file
      --stdin             Read body from stdin
      --plain             Send body as plain text (no markdown rendering)
      --html              Send body as raw HTML (no markdown rendering)
  -a, --attach strings    File path to attach (repeat for multiple)
      --reply-to string   Source Gmail message ID to reply to (derives To/Subject/threading)
      --reply-all         Include the source To/Cc as Cc on the reply (requires --reply-to)
      --no-quote          Reply without quoting the source message (requires --reply-to)
```

`--body`, `--stdin`, and `--file` are mutually exclusive; exactly one is required, except in reply mode where all three are optional (a bare reply is just the quote). `--plain` and `--html` are mutually exclusive.

Display names in `--to`/`--cc`/`--bcc` (e.g., `Alice <alice@example.com>`) are stripped; only the email address is sent. Edit the draft in Gmail to set display names.

With `--reply-to`, the draft is threaded onto the source conversation (`In-Reply-To` and `References` headers are set; `Draft.Message.ThreadId` is set to the source thread). `--to` and `--subject` are derived from the source (To = original From; Subject = `Re: <original>` with no double prefix). Explicit `--to`/`--cc`/`--subject` flags override the derived values. `--reply-all` populates Cc with the source To+Cc minus your authenticated account and any `--from` alias. The source message is quoted below your text by default (Gmail-style `On <date> <sender> wrote:` attribution; `gmail_quote` markup on HTML replies so Gmail collapses it natively); `--no-quote` suppresses the quote.

### gro calendar list

List all calendars the user has access to.

```
Usage: gro calendar list [flags]

Aliases: gro cal list

Flags:
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
```

### gro calendar get

Get the full details of a calendar event.

```
Usage: gro calendar get <event-id> [flags]

Aliases: gro cal get

Flags:
  -c, --calendar string   Calendar ID containing the event (default "primary")
```

### gro calendar today

Show all events for today.

```
Usage: gro calendar today [flags]

Aliases: gro cal today

Flags:
  -c, --calendar string   Calendar ID to query (default "primary")
```

### gro calendar week

Show all events for the current week (Monday to Sunday).

```
Usage: gro calendar week [flags]

Aliases: gro cal week

Flags:
  -c, --calendar string   Calendar ID to query (default "primary")
```

### gro calendar rsvp

Update your RSVP status on an event. Valid responses: accept, decline, tentative.

```
Usage: gro calendar rsvp <event-id> <accept|decline|tentative> [flags]

Aliases: gro cal rsvp

Flags:
  -c, --calendar string   Calendar ID containing the event (default "primary")
  -n, --dry-run           Preview without making changes
```

### gro calendar color

Set event color. Valid colors: lavender, sage, grape, flamingo, banana, tangerine, peacock, graphite, blueberry, basil, tomato (or IDs 1-11).

```
Usage: gro calendar color <event-id> <color> [flags]

Aliases: gro cal color

Flags:
  -c, --calendar string   Calendar ID containing the event (default "primary")
  -n, --dry-run           Preview without making changes
```

### gro contacts list

List all contacts sorted by last name.

```
Usage: gro contacts list [flags]

Aliases: gro ppl list

Flags:
  -m, --max int    Maximum number of contacts (default 10)
      --ids        Output only resource names (one per line, for piping)
```


### gro contacts search

Search contacts by name, email, phone, or organization.

```
Usage: gro contacts search <query> [flags]

Aliases: gro ppl search

Flags:
  -m, --max int    Maximum number of results (default 10)
      --ids        Output only resource names (one per line, for piping)
```


### gro contacts get

Get the full details of a specific contact.

```
Usage: gro contacts get <resource-name> [flags]

Aliases: gro ppl get

Flags:
```

### gro contacts groups

List all contact groups (labels).

```
Usage: gro contacts groups [flags]

Aliases: gro ppl groups

Flags:
  -m, --max int    Maximum number of groups (default 30)
```

### gro contacts star

Star contacts.

```
Usage: gro contacts star [contact-ids...] [flags]

Aliases: gro ppl star

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read contact IDs from stdin
      --query string   Search query to resolve contact IDs
```

### gro contacts unstar

Unstar contacts.

```
Usage: gro contacts unstar [contact-ids...] [flags]

Aliases: gro ppl unstar

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read contact IDs from stdin
      --query string   Search query to resolve contact IDs
```

### gro contacts add-to-group

Add contacts to a group.

```
Usage: gro contacts add-to-group <group-name> [contact-ids...] [flags]

Aliases: gro ppl add-to-group

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read contact IDs from stdin
      --query string   Search query to resolve contact IDs
```

### gro contacts remove-from-group

Remove contacts from a group.

```
Usage: gro contacts remove-from-group <group-name> [contact-ids...] [flags]

Aliases: gro ppl remove-from-group

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read contact IDs from stdin
      --query string   Search query to resolve contact IDs
```

### gro drive list

List files in Google Drive root or a specific folder.

```
Usage: gro drive list [folder-id] [flags]

Aliases: gro files list

Flags:
  -m, --max int      Maximum number of files (default 25)
  -t, --type string  Filter by type (document, spreadsheet, presentation, folder, pdf, image, video, audio)
      --ids          Output only file IDs (one per line, for piping)
      --my-drive     List from My Drive only
      --drive string List from specific shared drive (name or ID)
```

`--my-drive` and `--drive` are mutually exclusive.

### gro drive search

Search for files in Google Drive. By default, searches all drives you have access to.

```
Usage: gro drive search [query] [flags]

Aliases: gro files search

Flags:
  -n, --name                   Search filename only (not full-text content)
  -t, --type string            Filter by file type
      --owner string           Filter by owner (me, or email)
      --modified-after string  Modified after date (YYYY-MM-DD)
      --modified-before string Modified before date (YYYY-MM-DD)
      --in-folder string       Search within folder ID
      --ids                    Output only file IDs (one per line, for piping)
      --my-drive               Search only My Drive
      --drive string           Search specific shared drive (name or ID)
  -m, --max int                Maximum results (default 25)
```

`--my-drive` and `--drive` are mutually exclusive.

### gro drive get

Get detailed metadata for a file.

```
Usage: gro drive get <file-id> [flags]

Aliases: gro files get

Flags:
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
      --my-drive     Show My Drive only (default)
      --drive string Show tree from specific shared drive
```

### gro drive drives

List all shared drives accessible to you. Results are cached locally; use
`gro refresh drives` to force a refresh.

```
Usage: gro drive drives [flags]

Aliases: gro files drives

Flags:
      --refresh    Force refresh from API (deprecated; use 'gro refresh drives')
```

### gro refresh

Refresh gro's local cache. With no arguments, refreshes every cacheable
resource (today: `drives`). With `--status`, reports freshness without
fetching. With `--json`, emits a control-plane envelope.

```
Usage: gro refresh [resources...] [flags]

Flags:
      --status     Print cache freshness; no network calls
  -j, --json       Emit a JSON control-plane envelope
```

### gro drive star

Star files.

```
Usage: gro drive star [file-ids...] [flags]

Aliases: gro files star

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read file IDs from stdin
      --query string   Search query to resolve file IDs
```

### gro drive unstar

Unstar files.

```
Usage: gro drive unstar [file-ids...] [flags]

Aliases: gro files unstar

Flags:
  -n, --dry-run    Preview without making changes
      --stdin      Read file IDs from stdin
      --query string   Search query to resolve file IDs
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
| `oauth_client.json` | OAuth client JSON — deployment material, not a secret (from Google Cloud Console; legacy `credentials.json` is auto-migrated) |
| OS keyring (`google-readonly/default` → `oauth_token`) | OAuth access/refresh token — the only place the token is stored (legacy `token.json` is migrated in once, then removed) |
| `config.yml` | Non-secret config: `credential_ref`, `oauth_client_path`, `granted_scopes` (legacy `config.json` and the pre-MON-5371 `cache_ttl_hours` field are read once and ignored — cache TTL is now hard-coded per resource) |
| `cache/` | Cached API metadata for faster repeated lookups |

### Cache Settings

gro caches Drive metadata (like shared drive lists) to speed up repeated
commands. The cache TTL is hard-coded per resource (24 hours for the Drive
list) and is no longer user-configurable.

The cache lives in the OS cache directory — `$XDG_CACHE_HOME/google-readonly`
(or `~/.cache/google-readonly`) on Linux, `~/Library/Caches/google-readonly`
on macOS, `%LocalAppData%\google-readonly` on Windows — kept separate from
your config. A cache left by an older gro version (inside the config dir) is
relocated automatically on first use; `gro config clear --all` clears the
Drive metadata cache alongside config.

The cache is automatically repopulated when stale or after being cleared.

## Security

- This tool is **non-destructive by design** - no send, delete, or trash operations are possible
- Organizational operations (archive, label, star, RSVP, color, group management) are the most impactful actions available
- The OAuth token is stored only in the OS keyring via `cli-common/credstore` (macOS Keychain, Linux Secret Service, Windows Credential Manager); the opt-in encrypted-file backend is AES-encrypted with a passphrase from `GOOGLE_READONLY_KEYRING_PASSPHRASE`. Backend selection precedence: `--backend <name>` flag > `GOOGLE_READONLY_KEYRING_BACKEND` env var > `keyring.backend` config key > auto-detect
- The OAuth client JSON is deployment material (not a secret) and is never written to the keyring
- Credentials never leave your machine
- Zip extraction includes security safeguards (size limits, path traversal prevention)

## Troubleshooting

### "unable to read OAuth client JSON"

Ensure the OAuth client JSON exists (run `gro init`, or check `gro config show`):
```bash
ls -la ~/.config/google-readonly/oauth_client.json
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

### Token expires every 7 days

Your OAuth app is likely still in **"Testing"** mode. See [Publish Your OAuth App](#3-publish-your-oauth-app-recommended) in the setup guide. Apps in testing mode have tokens that expire after 7 days.

## License

MIT License - see [LICENSE](LICENSE) for details.

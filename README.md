# gro

A read-only command-line interface for Google services. Search, read, and view Gmail messages, threads, and attachments without any ability to modify, send, or delete data.

## Features

- **Read-only access** - Uses `gmail.readonly`, `calendar.readonly`, and `drive.readonly` OAuth scopes
- **Gmail support** - Search messages, read content, view threads, list labels, download attachments
- **JSON output** - Machine-readable output for scripting
- **Secure storage** - OAuth tokens stored in system keychain (macOS/Linux)

## Installation

### macOS

**Homebrew (recommended)**

```bash
brew install open-cli-collective/tap/gro
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
winget install OpenCLICollective.gro
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
sudo apt install gro
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
sudo dnf install gro
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
   - (Optional for future features) Enable: **Google Calendar API**, **Google Drive API**

### 2. Create OAuth Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. If prompted, configure the OAuth consent screen:
   - Choose **External** user type
   - Fill in required fields (app name, support email)
   - Add scopes:
     - `https://www.googleapis.com/auth/gmail.readonly`
     - `https://www.googleapis.com/auth/calendar.readonly` (optional)
     - `https://www.googleapis.com/auth/drive.readonly` (optional)
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

## Search Query Reference

gro supports all Gmail search operators:

| Operator | Example | Description |
|----------|---------|-------------|
| `from:` | `from:alice@example.com` | Messages from sender |
| `to:` | `to:bob@example.com` | Messages to recipient |
| `subject:` | `subject:meeting` | Subject contains word |
| `is:` | `is:unread`, `is:starred` | Message state |
| `has:` | `has:attachment` | Has attachment |
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
1. The Gmail API is enabled
2. Your email is added as a test user (for apps in testing mode)
3. The required scopes are added

## Future Features

- **Google Calendar** - Read events, list calendars
- **Google Drive** - List files, download content

## License

MIT License - see [LICENSE](LICENSE) for details.

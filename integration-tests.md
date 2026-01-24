# Integration Tests

Comprehensive integration test suite for gro. Tests are designed to work against any active Gmail account with standard inbox content.

## Test Environment Setup

### Prerequisites
- Valid OAuth credentials configured (`~/.config/google-readonly/credentials.json`)
- OAuth token (stored in system keychain or `~/.config/google-readonly/token.json`)
- Access to a Gmail account with:
  - At least some messages in the inbox
  - At least one email with attachments (for attachment tests)
  - At least one email thread with multiple messages

### Verification
```bash
ls ~/.config/google-readonly/credentials.json
gro config show  # Check configuration status
gro mail search "is:inbox" --max 1  # Quick connectivity check
```

---

## Version Command

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Print version | `gro --version` | Shows "gro <version>" |

---

## Config Commands

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Show config | `gro config show` | Shows credentials and token status |
| Test connectivity | `gro config test` | Shows "Successfully connected to Gmail API" |
| Clear token | `gro config clear` | Removes stored OAuth token |

---

## Search Operations

### Basic Search

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Search inbox | `gro mail search "is:inbox" --max 5` | Returns messages with ID, ThreadID, From, Subject, Date, Snippet |
| Search with default limit | `gro mail search "is:inbox"` | Returns up to 10 messages (default) |
| Custom result limit | `gro mail search "is:inbox" --max 3` | Returns exactly 3 messages |
| JSON output | `gro mail search "is:inbox" --max 2 --json` | Valid JSON array with message objects |
| No results | `gro mail search "xyznonexistent12345uniquequery67890"` | "No messages found." |
| Search unread | `gro mail search "is:unread" --max 5` | Returns unread messages (if any) |
| Search starred | `gro mail search "is:starred" --max 5` | Returns starred messages (if any) |

### Query Operators

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| From filter | `gro mail search "from:noreply" --max 3` | Messages from addresses containing "noreply" |
| Subject filter | `gro mail search "subject:welcome" --max 3` | Messages with "welcome" in subject |
| Has attachment | `gro mail search "has:attachment" --max 3` | Messages with attachments |
| Date range | `gro mail search "after:2024/01/01" --max 3` | Messages after date |
| Combined query | `gro mail search "is:inbox has:attachment" --max 3` | Inbox messages with attachments |
| Label filter | `gro mail search "label:inbox" --max 3` | Messages in inbox |

### Attachment Size and Type Search

Use graduated size thresholds to find attachments in inboxes of varying sizes.

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Larger than 10M | `gro mail search "has:attachment larger:10M" --max 3` | Large attachments (if any) |
| Larger than 5M | `gro mail search "has:attachment larger:5M" --max 3` | Medium-large attachments (if any) |
| Larger than 1M | `gro mail search "has:attachment larger:1M" --max 3` | Medium attachments (if any) |
| Larger than 500K | `gro mail search "has:attachment larger:500K" --max 3` | Small-medium attachments (if any) |
| Larger than 100K | `gro mail search "has:attachment larger:100K" --max 3` | Small attachments (if any) |
| Smaller than 1M | `gro mail search "has:attachment smaller:1M" --max 3` | Small attachments |
| PDF attachments | `gro mail search "filename:pdf" --max 3` | Messages with PDF files |
| Excel attachments | `gro mail search "filename:xlsx" --max 3` | Messages with Excel files |
| Zip attachments | `gro mail search "filename:zip" --max 3` | Messages with ZIP files |
| Combined size and type | `gro mail search "has:attachment filename:pdf larger:100K" --max 3` | PDFs over 100KB |

### JSON Validation

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| JSON has required fields | `gro mail search "is:inbox" --max 1 --json \| jq '.[0] \| keys'` | Contains: id, threadId, from, subject, date, snippet |
| JSON ID is string | `gro mail search "is:inbox" --max 1 --json \| jq -e '.[0].id \| type == "string"'` | Returns true |
| JSON ThreadID present | `gro mail search "is:inbox" --max 1 --json \| jq -e '.[0].threadId != null'` | Returns true |

---

## Read Operations

### Read Single Message

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Read by ID | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail read "$MSG_ID"` | Shows ID, From, To, Subject, Date, Body |
| Read JSON output | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail read "$MSG_ID" --json` | Valid JSON with body content |
| Non-existent message | `gro mail read "0000000000000000"` | Error: 404 or "not found" |
| Invalid message ID | `gro mail read "invalid-id-format"` | Error message |

### Read Content Verification

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Body included | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail read "$MSG_ID" --json \| jq -e '.body != null'` | Returns true |
| Headers present | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail read "$MSG_ID"` | Output contains "From:", "To:", "Subject:", "Date:" |

---

## Thread Operations

### Thread by Thread ID

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| View thread | `THREAD_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].threadId'); gro mail thread "$THREAD_ID"` | Shows "Thread contains N message(s)" and all messages |
| Thread JSON | `THREAD_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].threadId'); gro mail thread "$THREAD_ID" --json` | Valid JSON array of messages |

### Thread by Message ID

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Thread from message ID | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail thread "$MSG_ID"` | Shows thread containing that message |
| Thread message count | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail thread "$MSG_ID" --json \| jq 'length >= 1'` | Returns true (at least 1 message) |

### Error Cases

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Non-existent thread | `gro mail thread "0000000000000000"` | Error: 404 or "not found" |

---

## Labels Operations

### List Labels

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all labels | `gro mail labels` | Shows NAME, TYPE, TOTAL, UNREAD columns |
| Labels JSON output | `gro mail labels --json` | Valid JSON array with label objects |
| Labels JSON has fields | `gro mail labels --json \| jq -e '.[0] \| has("id", "name", "type")'` | Returns true |

### Label Display in Messages

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Search shows labels | `gro mail search "is:inbox" --max 1` | Output may include "Labels:" line if message has user labels |
| Search shows categories | `gro mail search "category:updates" --max 1` | Output may include "Categories: updates" |
| Search JSON has labels | `gro mail search "is:inbox" --max 1 --json \| jq '.[0] \| has("labels", "categories")'` | Returns true |
| Read shows labels | `MSG_ID=$(gro mail search "is:inbox" --max 1 --json \| jq -r '.[0].id'); gro mail read "$MSG_ID"` | Output may include "Labels:" and "Categories:" lines |

### Label-Based Search

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Search by label | `gro mail search "label:inbox" --max 3` | Returns inbox messages |
| Search by category | `gro mail search "category:updates" --max 3` | Returns updates category messages |
| Exclude category | `gro mail search "is:inbox -category:promotions" --max 3` | Returns inbox excluding promotions |
| Combined label search | `gro mail search "is:inbox -category:social -category:promotions" --max 3` | Inbox excluding social and promotions |

---

## Attachment Operations

### Setup: Find Message with Attachments
```bash
# Store a message ID with attachments for subsequent tests
ATTACHMENT_MSG_ID=$(gro mail search "has:attachment" --max 1 --json | jq -r '.[0].id')
```

### List Attachments

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List attachments | `gro mail attachments list "$ATTACHMENT_MSG_ID"` | Shows filename, type, size for each attachment |
| List JSON | `gro mail attachments list "$ATTACHMENT_MSG_ID" --json` | Valid JSON array with attachment metadata |
| No attachments | `MSG_ID=$(gro mail search "is:inbox -has:attachment" --max 1 --json \| jq -r '.[0].id'); gro mail attachments list "$MSG_ID"` | "No attachments found for message." |

### JSON Validation

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Attachment has filename | `gro mail attachments list "$ATTACHMENT_MSG_ID" --json \| jq -e '.[0].filename != null'` | Returns true |
| Attachment has mimeType | `gro mail attachments list "$ATTACHMENT_MSG_ID" --json \| jq -e '.[0].mimeType != null'` | Returns true |
| Attachment has size | `gro mail attachments list "$ATTACHMENT_MSG_ID" --json \| jq -e '.[0].size >= 0'` | Returns true |

### Download Attachments

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Download all (no flag) | `gro mail attachments download "$ATTACHMENT_MSG_ID"` | Error: "must specify --filename or --all" |
| Download all | `gro mail attachments download "$ATTACHMENT_MSG_ID" --all -o /tmp/gro-test` | Downloads all attachments, shows "Downloaded: ..." |
| Download specific file | `FILENAME=$(gro mail attachments list "$ATTACHMENT_MSG_ID" --json \| jq -r '.[0].filename'); gro mail attachments download "$ATTACHMENT_MSG_ID" -f "$FILENAME" -o /tmp/gro-test` | Downloads specific file |
| Non-existent filename | `gro mail attachments download "$ATTACHMENT_MSG_ID" -f "nonexistent-file-12345.xyz"` | Error: "attachment not found" |
| Verify file created | `ls /tmp/gro-test/` | Downloaded files exist |
| Verify file size > 0 | `stat -f%z /tmp/gro-test/* \| head -1` (macOS) | Non-zero file size |

### Zip Extraction (if zip attachment available)

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Find zip attachment | `ZIP_MSG_ID=$(gro mail search "has:attachment filename:zip" --max 1 --json \| jq -r '.[0].id')` | Message ID or null |
| Download and extract | `gro mail attachments download "$ZIP_MSG_ID" -f "*.zip" --extract -o /tmp/gro-zip-test` | Extracts to directory |
| Verify extraction | `ls /tmp/gro-zip-test/*/` | Extracted files present |

---

## Error Handling

| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Missing required arg (search) | `gro mail search` | Error: accepts 1 arg(s), received 0 |
| Missing required arg (read) | `gro mail read` | Error: accepts 1 arg(s), received 0 |
| Missing required arg (thread) | `gro mail thread` | Error: accepts 1 arg(s), received 0 |
| Missing required arg (attachments list) | `gro mail attachments list` | Error: accepts 1 arg(s), received 0 |
| Invalid subcommand | `gro invalid` | Error: unknown command |
| Help flag | `gro --help` | Shows usage information |
| Mail help | `gro mail --help` | Shows mail-specific help |
| Search help | `gro mail search --help` | Shows search-specific help |

---

## Output Format Consistency

### Text Output Structure

| Command Type | Expected Fields |
|--------------|-----------------|
| Search | ID, ThreadID, From, Subject, Date, Labels (if any), Categories (if any), Snippet, separator (---) |
| Read | ID, From, To, Subject, Date, Labels (if any), Categories (if any), "--- Body ---", body content |
| Thread | "Thread contains N message(s)", per-message: "=== Message X of Y ===", ID, From, To, Subject, Date, Labels, Categories, body |
| Labels | NAME, TYPE, TOTAL, UNREAD columns |
| Attachments List | "Found N attachment(s):", numbered list with filename, Type, Size |

### JSON Schema Validation

| Type | Required Fields |
|------|-----------------|
| Search result | id, threadId, from, subject, date, snippet, labels, categories |
| Message | id, threadId, from, to, subject, date, body, labels, categories |
| Attachment | filename, mimeType, size, partId |
| Label | id, name, type, messagesTotal, messagesUnread |

---

## End-to-End Workflows

### Workflow 1: Search -> Read -> Thread
```bash
# 1. Search for a message
MSG_ID=$(gro mail search "is:inbox" --max 1 --json | jq -r '.[0].id')

# 2. Read the full message
gro mail read "$MSG_ID"

# 3. View the full thread
gro mail thread "$MSG_ID"
```

### Workflow 2: Find and Download Attachments
```bash
# 1. Find message with attachments
ATTACHMENT_MSG_ID=$(gro mail search "has:attachment" --max 1 --json | jq -r '.[0].id')

# 2. List attachments
gro mail attachments list "$ATTACHMENT_MSG_ID"

# 3. Download all attachments
gro mail attachments download "$ATTACHMENT_MSG_ID" --all -o /tmp/gro-attachments

# 4. Verify downloads
ls -la /tmp/gro-attachments/
```

### Workflow 3: JSON Pipeline
```bash
# Extract all From addresses from recent inbox messages
gro mail search "is:inbox" --max 10 --json | jq -r '.[].from'

# Get message bodies from a thread
THREAD_ID=$(gro mail search "is:inbox" --max 1 --json | jq -r '.[0].threadId')
gro mail thread "$THREAD_ID" --json | jq -r '.[].body'
```

---

## Test Execution Checklist

### Setup
- [ ] Build latest: `make build`
- [ ] Verify credentials exist: `ls ~/.config/google-readonly/credentials.json`
- [ ] Quick connectivity test: `gro mail search "is:inbox" --max 1`

### Core Commands
- [ ] `gro --version`
- [ ] `gro config show`
- [ ] `gro config test`
- [ ] `gro mail search` with various queries
- [ ] `gro mail read` by message ID
- [ ] `gro mail thread` by thread ID
- [ ] `gro mail thread` by message ID
- [ ] `gro mail labels` (list all labels)

### Labels
- [ ] `gro mail labels` text output
- [ ] `gro mail labels --json` JSON output
- [ ] Search by label/category
- [ ] Labels/categories in message output

### Attachments
- [ ] `gro mail attachments list`
- [ ] `gro mail attachments download --all`
- [ ] `gro mail attachments download --filename`
- [ ] Zip extraction with `--extract`

### Attachment Size and Type Search
- [ ] `larger:` with graduated sizes (10M, 5M, 1M, 500K, 100K)
- [ ] `smaller:` filter
- [ ] `filename:` with pdf, xlsx, zip
- [ ] Combined `filename:` + `larger:` search

### Output Formats
- [ ] Text output for all commands
- [ ] JSON output for all commands
- [ ] JSON validates with jq

### Error Handling
- [ ] Missing arguments
- [ ] Invalid IDs
- [ ] Non-existent resources

### Cleanup
- [ ] Remove test downloads: `rm -rf /tmp/gro-test /tmp/gro-zip-test /tmp/gro-attachments`

---

## Adding New Tests

When adding new features or fixing bugs:

1. Add test cases to the appropriate section above
2. Include both happy path and error cases
3. Document any known limitations or edge cases
4. Update the "Test Execution Checklist" if needed

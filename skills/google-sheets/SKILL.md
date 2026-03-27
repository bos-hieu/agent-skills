---
name: google-sheets
description: Manage Google Sheets — create, read, write, append, and format spreadsheets via the Google Sheets REST API v4 with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to create, read, write, append, or manage Google Sheets:

1. Auto-detect Google credentials from config files or environment variables.
2. Let the user configure credentials with `--setup` if not yet configured.
3. Perform the requested operation (create, get, read, write, append, add-sheet, delete-sheet, clear).
4. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
5. **Never print raw access tokens or credentials — mask them in output.**
6. **Never save Google credentials to memory files or auto-memory.**

## Google Configuration

Credentials are shared with google-docs and discovered from these sources (highest priority first):

1. **Project config** (`.claude/google-config.yaml`) — project-specific settings
2. **Global config** (`~/.claude/google-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

### Authentication Methods

Two authentication methods are supported:

1. **Access Token** — An OAuth2 access token (e.g., from `gcloud auth print-access-token` or the Google OAuth playground).
2. **Service Account** — A path to a service account JSON key file. JWT signing is handled internally to obtain access tokens.

### Setting Up Credentials

```bash
# Setup with access token (project-level)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup \
  --access-token "ya29.xxx"

# Setup with service account file (global)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup --global \
  --service-account-file "/path/to/service-account.json"

# Show current configuration (masks token)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --show-config
```

### Config File Format

```yaml
# .claude/google-config.yaml or ~/.claude/google-config.yaml
auth_method: "access_token"
access_token: "ya29.xxx"
service_account_file: "/path/to/sa.json"
```

### Environment Variables

```
GOOGLE_ACCESS_TOKEN=ya29.xxx
GOOGLE_SERVICE_ACCOUNT_FILE=/path/to/sa.json
```

## Spreadsheet Operations

| Flag | Description | Example |
|------|-------------|---------|
| `--create` | Create a new spreadsheet | `--create --title "My Sheet"` |
| `--get <id>` | Get spreadsheet metadata | `--get abc123` |
| `--read <id>` | Read cell values | `--read abc123 --range "Sheet1!A1:D10"` |
| `--write <id>` | Write values to cells | `--write abc123 --range "Sheet1!A1" --values '[[1,2],[3,4]]'` |
| `--append <id>` | Append rows | `--append abc123 --range "Sheet1!A1" --values '[[5,6]]'` |
| `--add-sheet <id>` | Add a new sheet tab | `--add-sheet abc123 --sheet "NewTab"` |
| `--delete-sheet <id>` | Delete a sheet tab | `--delete-sheet abc123 --sheet "OldTab"` |
| `--clear <id>` | Clear a range of cells | `--clear abc123 --range "Sheet1!A1:D10"` |

## Content Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--title <title>` | Spreadsheet title (for --create) | `--title "Q1 Report"` |
| `--range <range>` | Cell range in A1 notation | `--range "Sheet1!A1:D10"` |
| `--sheet <name>` | Sheet tab name | `--sheet "Sheet1"` |
| `--values <json>` | JSON array of row arrays | `--values '[[1,2],[3,4]]'` |
| `--values-file <path>` | Read values JSON from file | `--values-file ./data.json` |
| `--format <fmt>` | Output format: text, json, csv (default text) | `--format csv` |
| `--rows <n>` | Max rows to display (default 50) | `--rows 100` |

## Config Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--setup` | Save Google credentials to config | `--setup --access-token "ya29.xxx"` |
| `--show-config` | Show current config (masks token) | `--show-config` |
| `--global` | Target global config instead of project | `--setup --global ...` |
| `--access-token <token>` | Google access token | `--access-token "ya29.xxx"` |
| `--service-account-file <path>` | Path to service account JSON | `--service-account-file "/path/to/sa.json"` |

## Examples

### Setup

```bash
# Configure with access token (project-level)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup \
  --access-token "ya29.a0AfH6SMB..."

# Configure with service account (global)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup --global \
  --service-account-file "/path/to/service-account.json"

# Verify config
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --show-config
```

### Creating Spreadsheets

```bash
# Create a new spreadsheet
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --create --title "Q1 Budget"
```

### Reading Data

```bash
# Get spreadsheet metadata (list all sheets)
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --get SPREADSHEET_ID

# Read a range of cells
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --read SPREADSHEET_ID --range "Sheet1!A1:D10"

# Read an entire sheet
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --read SPREADSHEET_ID --sheet "Sheet1"

# Read as CSV
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --read SPREADSHEET_ID --range "Sheet1!A1:D10" --format csv

# Read as JSON
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --read SPREADSHEET_ID --range "Sheet1!A1:D10" --format json
```

### Writing Data

```bash
# Write values to a range
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --write SPREADSHEET_ID \
  --range "Sheet1!A1" --values '[["Name","Age"],["Alice",30],["Bob",25]]'

# Write values from a file
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --write SPREADSHEET_ID \
  --range "Sheet1!A1" --values-file ./data.json

# Append rows to a sheet
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --append SPREADSHEET_ID \
  --range "Sheet1!A1" --values '[["Charlie",35],["Diana",28]]'
```

### Managing Sheets

```bash
# Add a new sheet tab
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --add-sheet SPREADSHEET_ID --sheet "Summary"

# Delete a sheet tab
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --delete-sheet SPREADSHEET_ID --sheet "OldData"

# Clear a range
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --clear SPREADSHEET_ID --range "Sheet1!A1:D10"
```

## Security Notes

- Access tokens and service account details are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save Google credentials to auto-memory or memory files.**
- Config is shared with google-docs at `.claude/google-config.yaml`.
- Do not commit config files with real credentials — `.claude/google-config.yaml` should be gitignored.

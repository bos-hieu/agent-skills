---
name: google-drive
description: Manage Google Drive files — list, upload, download, search, share, and organize files and folders. Supports Google Drive REST API v3 with persistent config for credentials shared with google-docs.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to list, upload, download, search, share, or manage Google Drive files and folders:

1. Auto-detect Google credentials from config files or environment variables.
2. Let the user configure credentials with `--setup` if not yet configured.
3. Perform the requested operation (list, get, download, upload, mkdir, delete, search, share).
4. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
5. **Never print raw access tokens or credentials — mask them in output.**
6. **Never save Google credentials to memory files or auto-memory.**

## Google Drive Configuration

Credentials are shared with the google-docs skill and discovered from these sources (highest priority first):

1. **Project config** (`.claude/google-config.yaml`) — project-specific settings
2. **Global config** (`~/.claude/google-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

### Setting Up Credentials

```bash
# Setup with access token (project-level)
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup \
  --access-token "ya29.xxx"

# Setup with service account file (project-level)
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup \
  --service-account-file "/path/to/sa.json"

# Save to global config (shared across projects)
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup --global \
  --access-token "ya29.xxx"

# Show current configuration (masks token)
go run ${CLAUDE_SKILL_DIR}/google_drive.go --show-config
```

### Config File Format

```yaml
# .claude/google-config.yaml or ~/.claude/google-config.yaml
auth_method: "access_token"  # or "service_account"
access_token: "ya29.xxx"
service_account_file: "/path/to/sa.json"
```

### Environment Variables

```
GOOGLE_ACCESS_TOKEN=ya29.xxx
GOOGLE_SERVICE_ACCOUNT_FILE=/path/to/sa.json
```

## File Operations

| Flag | Description | Example |
|------|-------------|---------|
| `--list` | List files | `--list` |
| `--get <file-id>` | Get file metadata | `--get 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms` |
| `--download <file-id>` | Download file content | `--download 1BxiMVs0XRA --output ./local.txt` |
| `--upload` | Upload a file | `--upload --file ./doc.pdf --title "My Doc"` |
| `--mkdir` | Create a folder | `--mkdir --title "New Folder"` |
| `--delete <file-id>` | Move file to trash | `--delete 1BxiMVs0XRA` |
| `--search <query>` | Search files by name/content | `--search "quarterly report"` |
| `--share <file-id>` | Share a file | `--share 1BxiMVs0XRA --email user@co.com --role writer` |

## Content Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--query <q>` | Drive search query (for --list) | `--query "mimeType='application/pdf'"` |
| `--folder <id>` | Folder ID (for --list, --upload, --mkdir) | `--folder 0BwwA4oUTeiV1TGRPeTVjaWRDY1E` |
| `--file <path>` | Local file path (for --upload) | `--file ./report.pdf` |
| `--title <name>` | File/folder name | `--title "Q4 Report"` |
| `--output <path>` | Output path (for --download) | `--output ./downloaded.pdf` |
| `--email <email>` | Email for sharing | `--email user@company.com` |
| `--role <role>` | Share role: reader, writer, commenter | `--role writer` |
| `--format <fmt>` | Output format: text, json (default: text) | `--format json` |
| `--rows <n>` | Max results (default 25) | `--rows 50` |

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
# Configure with access token
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup \
  --access-token "ya29.a0ARrdaM..."

# Configure with service account
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup \
  --service-account-file "/path/to/service-account.json"

# Verify config
go run ${CLAUDE_SKILL_DIR}/google_drive.go --show-config
```

### Listing Files

```bash
# List files in root
go run ${CLAUDE_SKILL_DIR}/google_drive.go --list

# List files in a folder
go run ${CLAUDE_SKILL_DIR}/google_drive.go --list --folder 0BwwA4oUTeiV1TGRPeTVjaWRDY1E

# List with custom query
go run ${CLAUDE_SKILL_DIR}/google_drive.go --list --query "mimeType='application/pdf'" --rows 10

# List as JSON
go run ${CLAUDE_SKILL_DIR}/google_drive.go --list --format json
```

### Downloading Files

```bash
# Download a file
go run ${CLAUDE_SKILL_DIR}/google_drive.go --download 1BxiMVs0XRA --output ./report.pdf

# Download to stdout (no --output)
go run ${CLAUDE_SKILL_DIR}/google_drive.go --download 1BxiMVs0XRA
```

### Uploading Files

```bash
# Upload a file
go run ${CLAUDE_SKILL_DIR}/google_drive.go --upload --file ./report.pdf --title "Q4 Report"

# Upload to a specific folder
go run ${CLAUDE_SKILL_DIR}/google_drive.go --upload --file ./doc.txt --title "Notes" --folder 0BwwA4oUTeiV1TGRPeTVjaWRDY1E
```

### Creating Folders

```bash
# Create a folder
go run ${CLAUDE_SKILL_DIR}/google_drive.go --mkdir --title "Project Files"

# Create a subfolder
go run ${CLAUDE_SKILL_DIR}/google_drive.go --mkdir --title "Designs" --folder 0BwwA4oUTeiV1TGRPeTVjaWRDY1E
```

### Searching

```bash
# Search by name
go run ${CLAUDE_SKILL_DIR}/google_drive.go --search "quarterly report"

# Search with max results
go run ${CLAUDE_SKILL_DIR}/google_drive.go --search "meeting notes" --rows 10
```

### Sharing

```bash
# Share with a user as writer
go run ${CLAUDE_SKILL_DIR}/google_drive.go --share 1BxiMVs0XRA --email user@company.com --role writer

# Share as reader
go run ${CLAUDE_SKILL_DIR}/google_drive.go --share 1BxiMVs0XRA --email viewer@company.com --role reader
```

### Other Operations

```bash
# Get file metadata
go run ${CLAUDE_SKILL_DIR}/google_drive.go --get 1BxiMVs0XRA

# Delete (trash) a file
go run ${CLAUDE_SKILL_DIR}/google_drive.go --delete 1BxiMVs0XRA
```

## Security Notes

- Access tokens are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save Google credentials to auto-memory or memory files.**
- Config is shared with google-docs skill at `.claude/google-config.yaml`.
- Do not commit config files with real credentials — `.claude/google-config.yaml` should be gitignored.

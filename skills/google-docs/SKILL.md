---
name: google-docs
description: Manage Google Docs — create, read, append, prepend, find-and-replace, and list documents. Supports Google Docs REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to create, read, edit, search, or manage Google Docs:

1. Auto-detect Google credentials from config files or environment variables.
2. Let the user configure credentials with `--setup` if not yet configured.
3. Perform the requested operation (create, get, append, prepend, replace, list).
4. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
5. **Never print raw API tokens — mask credentials in output.**
6. **Never save Google credentials to memory files or auto-memory.**

## Google Configuration

Credentials are discovered from three sources (highest priority first):

1. **Project config** (`.claude/google-config.yaml`) — project-specific settings
2. **Global config** (`~/.claude/google-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

### Authentication Methods

Two authentication methods are supported:

1. **Access Token** — An OAuth2 access token (e.g., from `gcloud auth print-access-token` or the Google OAuth playground).
2. **Service Account** — A path to a service account JSON key file. JWT signing is handled internally to obtain access tokens.

### Setting Up Credentials

```bash
# Setup with access token — saves to project config
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup \
  --access-token "ya29.xxx"

# Setup with service account file — saves to project config
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup \
  --service-account-file "/path/to/service-account.json"

# Save to global config (shared across projects)
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup --global \
  --access-token "ya29.xxx"

# Show current configuration (masks token)
go run ${CLAUDE_SKILL_DIR}/google_docs.go --show-config
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

## Document Operations

| Flag | Description | Example |
|------|-------------|---------|
| `--create` | Create a new document | `--create --title "My Document"` |
| `--get <doc-id>` | Get document content as plain text | `--get 1BxiMVs0XRA5nFMdKvBdBZjgmUii_` |
| `--append <doc-id>` | Append text to a document | `--append 1BxiMVs0 --body "New paragraph"` |
| `--prepend <doc-id>` | Prepend text to a document | `--prepend 1BxiMVs0 --body "Introduction"` |
| `--replace <doc-id>` | Find and replace text | `--replace 1BxiMVs0 --find "old" --replace-with "new"` |
| `--list` | List recent documents | `--list` |

## Content Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--title <title>` | Document title (for --create) | `--title "Meeting Notes"` |
| `--body <text>` | Text content to append/prepend | `--body "Hello, world!"` |
| `--body-file <path>` | Read text content from a file | `--body-file ./notes.txt` |
| `--find <text>` | Text to find (for --replace) | `--find "TODO"` |
| `--replace-with <text>` | Replacement text (for --replace) | `--replace-with "DONE"` |
| `--format <format>` | Output format: text, json (default: text) | `--format json` |
| `--rows <n>` | Max results for list (default 25) | `--rows 50` |

## Config Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--setup` | Save Google credentials to config | `--setup --access-token "ya29.xxx"` |
| `--show-config` | Show current config (masks token) | `--show-config` |
| `--global` | Target global config instead of project | `--setup --global ...` |
| `--access-token <token>` | OAuth2 access token | `--access-token "ya29.xxx"` |
| `--service-account-file <path>` | Path to service account JSON key file | `--service-account-file "/path/to/sa.json"` |

## Examples

### Setup

```bash
# Configure with access token (project-level)
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup \
  --access-token "$(gcloud auth print-access-token)"

# Configure with service account (global)
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup --global \
  --service-account-file "/path/to/service-account.json"

# Verify config
go run ${CLAUDE_SKILL_DIR}/google_docs.go --show-config
```

### Creating Documents

```bash
# Create a new document
go run ${CLAUDE_SKILL_DIR}/google_docs.go --create --title "Sprint 42 Notes"
```

### Reading Documents

```bash
# Get document content as plain text
go run ${CLAUDE_SKILL_DIR}/google_docs.go --get 1BxiMVs0XRA5nFMdKvBdBZjgmUii_

# Get document content as JSON
go run ${CLAUDE_SKILL_DIR}/google_docs.go --get 1BxiMVs0XRA5nFMdKvBdBZjgmUii_ --format json
```

### Editing Documents

```bash
# Append text to a document
go run ${CLAUDE_SKILL_DIR}/google_docs.go --append 1BxiMVs0 \
  --body "This text is appended at the end."

# Prepend text to a document
go run ${CLAUDE_SKILL_DIR}/google_docs.go --prepend 1BxiMVs0 \
  --body "This text appears at the beginning."

# Append text from a file
go run ${CLAUDE_SKILL_DIR}/google_docs.go --append 1BxiMVs0 \
  --body-file ./additional-notes.txt

# Find and replace text
go run ${CLAUDE_SKILL_DIR}/google_docs.go --replace 1BxiMVs0 \
  --find "DRAFT" --replace-with "FINAL"
```

### Listing Documents

```bash
# List recent documents
go run ${CLAUDE_SKILL_DIR}/google_docs.go --list

# List with limit
go run ${CLAUDE_SKILL_DIR}/google_docs.go --list --rows 10

# List as JSON
go run ${CLAUDE_SKILL_DIR}/google_docs.go --list --format json
```

## Security Notes

- Access tokens are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save Google credentials to auto-memory or memory files.**
- Do not commit config files with real credentials — `.claude/google-config.yaml` should be gitignored.

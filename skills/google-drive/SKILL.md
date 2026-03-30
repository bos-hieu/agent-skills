---
name: google-drive
description: Manage Google Drive files — list, upload, download, search, share, and organize files and folders. Supports Google Drive REST API v3 with persistent config for credentials shared with google-docs.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Google Drive files, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw tokens. Never save credentials to memory files or auto-memory.**

## Configuration

Shares config with google-docs skill. Credentials discovered from (highest priority first):
1. Project config: `.claude/google-config.yaml`
2. Global config: `~/.claude/google-config.yaml`
3. Environment variables: `GOOGLE_ACCESS_TOKEN`, `GOOGLE_SERVICE_ACCOUNT_FILE`

```bash
# Setup
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup --access-token "ya29.xxx"
go run ${CLAUDE_SKILL_DIR}/google_drive.go --setup --service-account-file "/path/to/sa.json"
# Use --global for ~/.claude/. Use --show-config to verify.
```

## Operations

| Flag | Description |
|------|-------------|
| `--list` | List files (with optional `--folder <id>`, `--query <q>`) |
| `--get <file-id>` | Get file metadata |
| `--download <file-id>` | Download file (with optional `--output <path>`) |
| `--upload` | Upload file (with `--file <path>`, `--title <name>`, optional `--folder <id>`) |
| `--mkdir` | Create folder (with `--title`, optional `--folder <id>`) |
| `--delete <file-id>` | Move file to trash |
| `--search <query>` | Search files by name/content |
| `--share <file-id>` | Share file (with `--email`, `--role` reader/writer/commenter) |
| `--format <text\|json>` | Output format (default: text) |
| `--rows <n>` | Max results (default 25) |

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Config shared with google-docs and google-sheets. Gitignore `.claude/google-config.yaml`.

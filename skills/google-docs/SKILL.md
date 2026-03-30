---
name: google-docs
description: Manage Google Docs — create, read, append, prepend, find-and-replace, and list documents. Supports Google Docs REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Google Docs, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw tokens. Never save credentials to memory files or auto-memory.**

## Configuration

Credentials discovered from (highest priority first):
1. Project config: `.claude/google-config.yaml`
2. Global config: `~/.claude/google-config.yaml`
3. Environment variables: `GOOGLE_ACCESS_TOKEN`, `GOOGLE_SERVICE_ACCOUNT_FILE`

Auth methods: **Access Token** (OAuth2, e.g. `gcloud auth print-access-token`) or **Service Account** (JSON key file, JWT handled internally).

```yaml
# .claude/google-config.yaml
auth_method: "access_token"  # or "service_account"
access_token: "ya29.xxx"
service_account_file: "/path/to/sa.json"
```

```bash
# Setup
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup --access-token "ya29.xxx"
go run ${CLAUDE_SKILL_DIR}/google_docs.go --setup --service-account-file "/path/to/sa.json"
# Use --global for ~/.claude/. Use --show-config to verify.
```

## Operations

| Flag | Description |
|------|-------------|
| `--create` | Create document (with `--title`) |
| `--get <doc-id>` | Get document content as plain text |
| `--append <doc-id>` | Append text (with `--body` or `--body-file`) |
| `--prepend <doc-id>` | Prepend text (with `--body` or `--body-file`) |
| `--replace <doc-id>` | Find and replace (with `--find`, `--replace-with`) |
| `--list` | List recent documents |
| `--format <text\|json>` | Output format (default: text) |
| `--rows <n>` | Max results for list (default 25) |

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Config shared with google-drive and google-sheets skills. Gitignore `.claude/google-config.yaml`.

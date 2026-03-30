---
name: google-sheets
description: Manage Google Sheets — create, read, write, append, and format spreadsheets via the Google Sheets REST API v4 with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Google Sheets, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw tokens. Never save credentials to memory files or auto-memory.**

## Configuration

Shares config with google-docs skill. Credentials discovered from (highest priority first):
1. Project config: `.claude/google-config.yaml`
2. Global config: `~/.claude/google-config.yaml`
3. Environment variables: `GOOGLE_ACCESS_TOKEN`, `GOOGLE_SERVICE_ACCOUNT_FILE`

```bash
# Setup
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup --access-token "ya29.xxx"
go run ${CLAUDE_SKILL_DIR}/google_sheets.go --setup --service-account-file "/path/to/sa.json"
# Use --global for ~/.claude/. Use --show-config to verify.
```

## Operations

| Flag | Description |
|------|-------------|
| `--create` | Create spreadsheet (with `--title`) |
| `--get <id>` | Get spreadsheet metadata |
| `--read <id>` | Read cell values (with `--range` or `--sheet`) |
| `--write <id>` | Write values (with `--range`, `--values` JSON or `--values-file`) |
| `--append <id>` | Append rows (with `--range`, `--values` or `--values-file`) |
| `--add-sheet <id>` | Add sheet tab (with `--sheet`) |
| `--delete-sheet <id>` | Delete sheet tab (with `--sheet`) |
| `--clear <id>` | Clear cell range (with `--range`) |
| `--format <text\|json\|csv>` | Output format (default: text) |
| `--rows <n>` | Max rows to display (default 50) |

Values format: JSON array of row arrays, e.g. `'[["Name","Age"],["Alice",30]]'`

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Config shared with google-docs and google-drive. Gitignore `.claude/google-config.yaml`.

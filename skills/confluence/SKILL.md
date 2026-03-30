---
name: confluence
description: Manage Confluence pages — create, read, update, search, and delete wiki pages. Supports Confluence Cloud via REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Confluence wiki pages, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw API tokens. Never save credentials to memory files or auto-memory.**

## Configuration

Credentials discovered from (highest priority first):
1. Project config: `.claude/confluence-config.yaml`
2. Global config: `~/.claude/confluence-config.yaml`
3. Environment variables: `CONFLUENCE_BASE_URL`, `CONFLUENCE_EMAIL`, `CONFLUENCE_API_TOKEN`

```yaml
# .claude/confluence-config.yaml
base_url: "https://yourcompany.atlassian.net"
email: "you@company.com"
api_token: "your-api-token"
```

```bash
# Setup credentials
go run ${CLAUDE_SKILL_DIR}/confluence.go --setup \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" --api-token "your-api-token"

# Use --global to save to ~/.claude/ instead of .claude/
# Use --show-config to verify (masks token)
```

## Operations

| Flag | Description |
|------|-------------|
| `--create` | Create page (requires `--space`, `--title`, `--body` or `--body-file`) |
| `--get <page-id>` | Get page by ID |
| `--update <page-id>` | Update page (with `--title`, `--body`, or `--body-file`) |
| `--delete <page-id>` | Delete page |
| `--search <CQL>` | Search pages using CQL |
| `--comment <page-id>` | Add comment (with `--body`) |
| `--children <page-id>` | List child pages |
| `--spaces` | List all spaces |
| `--parent <page-id>` | Parent page ID (for `--create`) |
| `--format <text\|json>` | Output format (default: text) |
| `--rows <n>` | Max results (default 25) |

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Generate tokens at: https://id.atlassian.com/manage-profile/security/api-tokens
- Gitignore `.claude/confluence-config.yaml`.

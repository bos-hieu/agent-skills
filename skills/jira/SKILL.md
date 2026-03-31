---
name: jira
description: Manage Jira issues — create, read, update, transition, search, comment, and assign issues. Supports Jira Cloud via REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Jira issues, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw API tokens. Never save credentials to memory files or auto-memory.**

## Configuration

Credentials discovered from (highest priority first):
1. Project config: `.claude/jira-config.yaml`
2. Global config: `~/.claude/jira-config.yaml`
3. Environment variables: `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN`

```yaml
# .claude/jira-config.yaml
base_url: "https://yourcompany.atlassian.net"
email: "you@company.com"
api_token: "your-api-token"
```

```bash
# Setup
go run ${CLAUDE_SKILL_DIR}/jira.go --setup \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" --api-token "your-api-token"
# Use --global for ~/.claude/. Use --show-config to verify.
```

## Operations

| Flag | Description |
|------|-------------|
| `--create` | Create issue (requires `--project`, `--type`, `--summary`) |
| `--get <issue-key>` | Get issue by key |
| `--update <issue-key>` | Update issue fields |
| `--transition <issue-key>` | Transition status (with `--status`) |
| `--comment <issue-key>` | Add comment (with `--body`) |
| `--search <JQL>` | Search issues using JQL |
| `--assign <issue-key>` | Assign issue (with `--assignee`) |
| `--projects` | List all projects |

## Content Flags

| Flag | Description |
|------|-------------|
| `--project <key>` | Project key |
| `--type <name>` | Issue type (Task, Bug, Subtask, etc.) |
| `--summary <text>` | Issue summary |
| `--description <text>` | Issue description |
| `--priority <name>` | Priority name |
| `--assignee <email>` | Assignee email or account ID |
| `--labels <csv>` | Comma-separated labels |
| `--parent <issue-key>` | Parent issue (for subtasks) |
| `--status <name>` | Target status (for `--transition`) |
| `--format <text\|json>` | Output format (default: text) |
| `--rows <n>` | Max results (default 25) |

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Generate tokens at: https://id.atlassian.com/manage-profile/security/api-tokens
- Gitignore `.claude/jira-config.yaml`.

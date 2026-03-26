---
name: jira
description: Manage Jira issues — create, read, update, transition, search, comment, and assign issues. Supports Jira Cloud via REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to create, read, edit, search, transition, or manage Jira issues:

1. Auto-detect Jira credentials from config files or environment variables.
2. Let the user configure credentials with `--setup` if not yet configured.
3. Perform the requested operation (create, get, update, transition, search, comment, assign, list projects).
4. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
5. **Never print raw API tokens — mask credentials in output.**
6. **Never save Jira credentials to memory files or auto-memory.**

## Jira Configuration

Credentials are discovered from two sources (highest priority first):

1. **Project config** (`.claude/jira-config.yaml`) — project-specific settings
2. **Global config** (`~/.claude/jira-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

### Setting Up Credentials

```bash
# Interactive setup — saves to project config
go run ${CLAUDE_SKILL_DIR}/jira.go --setup \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" \
  --api-token "your-api-token"

# Save to global config (shared across projects)
go run ${CLAUDE_SKILL_DIR}/jira.go --setup --global \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" \
  --api-token "your-api-token"

# Show current configuration (masks token)
go run ${CLAUDE_SKILL_DIR}/jira.go --show-config
```

### Config File Format

```yaml
# .claude/jira-config.yaml or ~/.claude/jira-config.yaml
base_url: "https://yourcompany.atlassian.net"
email: "you@company.com"
api_token: "your-api-token"
```

### Environment Variables

```
JIRA_BASE_URL=https://yourcompany.atlassian.net
JIRA_EMAIL=you@company.com
JIRA_API_TOKEN=your-api-token
```

## Issue Operations

| Flag | Description | Example |
|------|-------------|---------|
| `--create` | Create a new issue | `--create --project DEV --type Task --summary "Fix bug" --description "Details"` |
| `--get <issue-key>` | Get an issue by key | `--get DEV-123` |
| `--update <issue-key>` | Update an issue | `--update DEV-123 --summary "New title" --priority High` |
| `--transition <issue-key>` | Transition issue status | `--transition DEV-123 --status "Done"` |
| `--comment <issue-key>` | Add a comment to an issue | `--comment DEV-123 --body "Looks good!"` |
| `--search <JQL>` | Search issues using JQL | `--search "project=DEV AND status='In Progress'"` |
| `--assign <issue-key>` | Assign an issue | `--assign DEV-123 --assignee "user@company.com"` |
| `--projects` | List all projects | `--projects` |

## Content Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--project <key>` | Project key (required for --create) | `--project DEV` |
| `--type <name>` | Issue type (required for --create) | `--type Task` |
| `--summary <text>` | Issue summary | `--summary "Fix login bug"` |
| `--description <text>` | Issue description | `--description "Steps to reproduce..."` |
| `--priority <name>` | Priority name | `--priority High` |
| `--assignee <email>` | Assignee email or account ID | `--assignee "user@company.com"` |
| `--labels <csv>` | Comma-separated labels | `--labels "bug,frontend"` |
| `--parent <issue-key>` | Parent issue key (for subtasks/child issues) | `--parent DEV-100` |
| `--status <name>` | Target status (for --transition) | `--status "In Progress"` |
| `--body <text>` | Comment body (for --comment) | `--body "LGTM"` |
| `--format <format>` | Output format: text, json (default: text) | `--format json` |
| `--rows <n>` | Max results for search/list (default 25) | `--rows 50` |

## Config Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--setup` | Save Jira credentials to config | `--setup --base-url "..." --email "..." --api-token "..."` |
| `--show-config` | Show current config (masks token) | `--show-config` |
| `--global` | Target global config instead of project | `--setup --global ...` |
| `--base-url <url>` | Jira base URL | `--base-url "https://co.atlassian.net"` |
| `--email <email>` | Jira user email | `--email "user@co.com"` |
| `--api-token <token>` | Jira API token | `--api-token "token"` |

## Examples

### Setup

```bash
# Configure credentials (project-level)
go run ${CLAUDE_SKILL_DIR}/jira.go --setup \
  --base-url "https://mycompany.atlassian.net" \
  --email "dev@mycompany.com" \
  --api-token "ATATT3xFfGF0..."

# Verify config
go run ${CLAUDE_SKILL_DIR}/jira.go --show-config
```

### Creating Issues

```bash
# Create a task
go run ${CLAUDE_SKILL_DIR}/jira.go --create \
  --project DEV --type Task --summary "Implement login page" \
  --description "Build the login page with OAuth support" \
  --priority High --labels "frontend,auth"

# Create a subtask under an existing issue
go run ${CLAUDE_SKILL_DIR}/jira.go --create \
  --project DEV --type Subtask --summary "Add OAuth callback" \
  --parent DEV-100

# Create a bug
go run ${CLAUDE_SKILL_DIR}/jira.go --create \
  --project DEV --type Bug --summary "Login fails on Safari" \
  --description "Steps to reproduce..." --priority Critical
```

### Reading Issues

```bash
# Get an issue by key
go run ${CLAUDE_SKILL_DIR}/jira.go --get DEV-123

# Get issue details as JSON
go run ${CLAUDE_SKILL_DIR}/jira.go --get DEV-123 --format json
```

### Updating Issues

```bash
# Update summary and priority
go run ${CLAUDE_SKILL_DIR}/jira.go --update DEV-123 \
  --summary "Updated title" --priority Medium

# Update assignee and labels
go run ${CLAUDE_SKILL_DIR}/jira.go --update DEV-123 \
  --assignee "dev@company.com" --labels "backend,api"
```

### Transitioning Issues

```bash
# Move issue to In Progress
go run ${CLAUDE_SKILL_DIR}/jira.go --transition DEV-123 --status "In Progress"

# Mark issue as Done
go run ${CLAUDE_SKILL_DIR}/jira.go --transition DEV-123 --status "Done"
```

### Searching

```bash
# Search by project
go run ${CLAUDE_SKILL_DIR}/jira.go --search "project=DEV AND status='To Do'" --rows 10

# Search assigned issues
go run ${CLAUDE_SKILL_DIR}/jira.go --search "assignee=currentUser() AND status!='Done'" --rows 50

# Full-text search
go run ${CLAUDE_SKILL_DIR}/jira.go --search "text~'login bug'" --rows 20
```

### Other Operations

```bash
# List all projects
go run ${CLAUDE_SKILL_DIR}/jira.go --projects

# Add a comment
go run ${CLAUDE_SKILL_DIR}/jira.go --comment DEV-123 --body "Reviewed and approved."

# Assign an issue
go run ${CLAUDE_SKILL_DIR}/jira.go --assign DEV-123 --assignee "dev@company.com"
```

## Security Notes

- API tokens are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save Jira credentials to auto-memory or memory files.**
- Generate API tokens at: https://id.atlassian.com/manage-profile/security/api-tokens
- Do not commit config files with real credentials — `.claude/jira-config.yaml` should be gitignored.

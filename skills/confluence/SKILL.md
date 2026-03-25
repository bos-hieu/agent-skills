---
name: confluence
description: Manage Confluence pages — create, read, update, search, and delete wiki pages. Supports Confluence Cloud via REST API with persistent config for credentials.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to create, read, edit, search, or manage Confluence wiki pages:

1. Auto-detect Confluence credentials from config files or environment variables.
2. Let the user configure credentials with `--setup` if not yet configured.
3. Perform the requested operation (create, read, update, search, delete, comment, list spaces).
4. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
5. **Never print raw API tokens — mask credentials in output.**
6. **Never save Confluence credentials to memory files or auto-memory.**

## Confluence Configuration

Credentials are discovered from two sources (highest priority first):

1. **Project config** (`.claude/confluence-config.yaml`) — project-specific settings
2. **Global config** (`~/.claude/confluence-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

### Setting Up Credentials

```bash
# Interactive setup — saves to project config
go run ${CLAUDE_SKILL_DIR}/confluence.go --setup \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" \
  --api-token "your-api-token"

# Save to global config (shared across projects)
go run ${CLAUDE_SKILL_DIR}/confluence.go --setup --global \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" \
  --api-token "your-api-token"

# Show current configuration (masks token)
go run ${CLAUDE_SKILL_DIR}/confluence.go --show-config
```

### Config File Format

```yaml
# .claude/confluence-config.yaml or ~/.claude/confluence-config.yaml
base_url: "https://yourcompany.atlassian.net"
email: "you@company.com"
api_token: "your-api-token"
```

### Environment Variables

```
CONFLUENCE_BASE_URL=https://yourcompany.atlassian.net
CONFLUENCE_EMAIL=you@company.com
CONFLUENCE_API_TOKEN=your-api-token
```

## Page Operations

| Flag | Description | Example |
|------|-------------|---------|
| `--create` | Create a new page | `--create --space DEV --title "My Page" --body "Content here"` |
| `--get <page-id>` | Get a page by ID | `--get 12345` |
| `--update <page-id>` | Update a page by ID | `--update 12345 --title "New Title" --body "New content"` |
| `--delete <page-id>` | Delete a page by ID | `--delete 12345` |
| `--search <CQL>` | Search pages using CQL | `--search "space=DEV AND title~\"API docs\""` |
| `--comment <page-id>` | Add a comment to a page | `--comment 12345 --body "Looks good!"` |
| `--children <page-id>` | List child pages | `--children 12345` |
| `--spaces` | List all spaces | `--spaces` |

## Content Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--space <key>` | Space key (required for --create) | `--space DEV` |
| `--title <title>` | Page title | `--title "Meeting Notes"` |
| `--body <content>` | Page body in Confluence storage format (HTML) | `--body "<p>Hello</p>"` |
| `--body-file <path>` | Read page body from a file | `--body-file ./content.html` |
| `--parent <page-id>` | Parent page ID (for --create) | `--parent 12345` |
| `--format <format>` | Output format: text, json (default: text) | `--format json` |
| `--rows <n>` | Max results for search/list (default 25) | `--rows 50` |

## Config Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--setup` | Save Confluence credentials to config | `--setup --base-url "..." --email "..." --api-token "..."` |
| `--show-config` | Show current config (masks token) | `--show-config` |
| `--global` | Target global config instead of project | `--setup --global ...` |
| `--base-url <url>` | Confluence base URL | `--base-url "https://co.atlassian.net"` |
| `--email <email>` | Confluence user email | `--email "user@co.com"` |
| `--api-token <token>` | Confluence API token | `--api-token "token"` |

## Examples

### Setup

```bash
# Configure credentials (project-level)
go run ${CLAUDE_SKILL_DIR}/confluence.go --setup \
  --base-url "https://mycompany.atlassian.net" \
  --email "dev@mycompany.com" \
  --api-token "ATATT3xFfGF0..."

# Verify config
go run ${CLAUDE_SKILL_DIR}/confluence.go --show-config
```

### Creating Pages

```bash
# Create a simple page
go run ${CLAUDE_SKILL_DIR}/confluence.go --create \
  --space DEV --title "Sprint 42 Retrospective" \
  --body "<h2>What went well</h2><ul><li>Shipped on time</li></ul>"

# Create a child page under an existing page
go run ${CLAUDE_SKILL_DIR}/confluence.go --create \
  --space DEV --title "API Reference" \
  --parent 98765 --body-file ./api-docs.html

# Create a page with body from file
go run ${CLAUDE_SKILL_DIR}/confluence.go --create \
  --space TEAM --title "Onboarding Guide" \
  --body-file ./onboarding.html
```

### Reading Pages

```bash
# Get a page by ID
go run ${CLAUDE_SKILL_DIR}/confluence.go --get 12345

# Get page content as JSON
go run ${CLAUDE_SKILL_DIR}/confluence.go --get 12345 --format json
```

### Updating Pages

```bash
# Update page title and body
go run ${CLAUDE_SKILL_DIR}/confluence.go --update 12345 \
  --title "Updated Title" --body "<p>Updated content</p>"

# Update only the body from a file
go run ${CLAUDE_SKILL_DIR}/confluence.go --update 12345 \
  --body-file ./updated-content.html
```

### Searching

```bash
# Search by title
go run ${CLAUDE_SKILL_DIR}/confluence.go --search "title=\"Meeting Notes\""

# Search within a space
go run ${CLAUDE_SKILL_DIR}/confluence.go --search "space=DEV AND text~\"deployment\"" --rows 10

# Full-text search
go run ${CLAUDE_SKILL_DIR}/confluence.go --search "text~\"API endpoint\"" --rows 20
```

### Other Operations

```bash
# List all spaces
go run ${CLAUDE_SKILL_DIR}/confluence.go --spaces

# List child pages
go run ${CLAUDE_SKILL_DIR}/confluence.go --children 12345

# Add a comment
go run ${CLAUDE_SKILL_DIR}/confluence.go --comment 12345 --body "Reviewed and approved."

# Delete a page
go run ${CLAUDE_SKILL_DIR}/confluence.go --delete 12345
```

## Security Notes

- API tokens are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save Confluence credentials to auto-memory or memory files.**
- Generate API tokens at: https://id.atlassian.com/manage-profile/security/api-tokens
- Do not commit config files with real credentials — `.claude/confluence-config.yaml` should be gitignored.

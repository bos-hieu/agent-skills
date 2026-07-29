---
name: confluence
description: Manage Confluence pages and folders — create, read, update, move, search, and delete wiki content. Supports Confluence Cloud via REST API with persistent config for credentials.
allowed-tools: Bash(go build *), Bash(go run *), Bash(cat *), Bash(ls *)
---

When the user asks to manage Confluence wiki content, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw API tokens. Never save credentials to memory files or auto-memory.**

## Running it

Build once, then run the binary:

```bash
(cd "${CLAUDE_SKILL_DIR}/../.." && go build -o /tmp/confluence-skill ./skills/confluence)
/tmp/confluence-skill --spaces
```

Do **not** use `go run ${CLAUDE_SKILL_DIR}/confluence.go` — the module lives at the plugin
root, so Go resolves it from your current directory and fails with `no required module
provides package gopkg.in/yaml.v3` whenever you are working in another project.

Do **not** "fix" that with `go run -C`. It compiles, but `-C` also changes the working
directory of the program, so a project-local `.claude/confluence-config.yaml` is never
seen and the global config is used instead — silently publishing to the wrong site.
Building first and running the binary from the user's directory keeps config discovery correct.

## Configuration

Credentials discovered from (highest priority first):
1. Project config: `.claude/confluence-config.yaml`
2. Global config: `~/.claude/confluence-config.yaml`
3. Environment variables: `CONFLUENCE_BASE_URL`, `CONFLUENCE_EMAIL`, `CONFLUENCE_API_TOKEN`

```yaml
# .claude/confluence-config.yaml
base_url: "https://yourcompany.atlassian.net"   # site root, no /wiki suffix
email: "you@company.com"
api_token: "your-api-token"
```

```bash
/tmp/confluence-skill --setup \
  --base-url "https://yourcompany.atlassian.net" \
  --email "you@company.com" --api-token "your-api-token"

# Use --global to save to ~/.claude/ instead of .claude/
# Use --show-config to verify (masks token)
```

Ask the user to run `--setup` themselves with a `!` prefix so the token stays out of the transcript.

## Operations

| Flag | Description |
|------|-------------|
| `--create` | Create page (requires `--space`, `--title`, `--body` or `--body-file`) |
| `--create-folder` | Create folder (requires `--space`, `--title`) |
| `--get <id>` | Get page or folder by ID |
| `--update <id>` | Update page (with `--title`, `--body`/`--body-file`, and/or `--parent` to move) |
| `--delete <id>` | Delete page or folder |
| `--search <CQL>` | Search pages using CQL |
| `--comment <page-id>` | Add comment (with `--body`) |
| `--children <id>` | List direct children of a page or folder |
| `--spaces` | List all spaces |
| `--parent <id>` | Parent page or folder ID (for `--create`, `--create-folder`, `--update`) |
| `--format <text\|json>` | Output format (default: text) |
| `--rows <n>` | Max results (default 25) |

## Pages, folders, and moving

Folders group content without holding any of their own — use them to separate reference
docs from important pages. They are a distinct content type: they never appear in `--search`
results, which cover pages only. Use `--children` to see inside one.

Moving is an update with `--parent`. The API only honours `parentId` on a full update, so
resend the body or it will be wiped:

```bash
/tmp/confluence-skill --update 12345 --parent 67890 --body-file page.html
```

Body is Confluence **storage format** (XHTML), not Markdown. It must be well-formed.
Macros are `<ac:structured-macro>` elements; put code inside `<![CDATA[...]]>`.
Before writing a page, read a neighbouring one with `--get` and match its conventions.

## Security

- Tokens always shown as `***`. Config files created with `0600` permissions.
- Generate tokens at: https://id.atlassian.com/manage-profile/security/api-tokens
- Gitignore `.claude/confluence-config.yaml`.

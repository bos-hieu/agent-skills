# Agent Skills

A collection of reusable skills for AI coding agents.

## Available Skills

| Skill | Description |
|---|---|
| **browser-test** | Run and debug Playwright browser tests (API, E2E, smoke) |
| **confluence** | Manage Confluence pages and folders — create, read, update, move, search, delete |
| **database** | Query PostgreSQL, MySQL, SQLite, and MongoDB with persistent config |
| **generating-claude-instructions** | Generate a CLAUDE.md file by exploring actual source code |
| **go-issue-solver** | Analyze and solve Go codebase issues from bug reports or logs |
| **google-docs** | Manage Google Docs — create, read, append, replace |
| **google-drive** | Manage Google Drive — list, upload, download, search, share |
| **google-sheets** | Manage Google Sheets — create, read, write, append |
| **jira** | Manage Jira issues — create, read, update, transition, search |
| **openclaw-docker-setup** | Set up and manage OpenClaw gateway instances in Docker |
| **pdf** | Read PDF files — extract text, search keywords, page ranges |
| **prd** | Generate PRDs, user stories, feature specs, release notes |
| **ssh** | Manage SSH connections with bastion support, tunneling, file transfer |
| **test-cases** | Generate comprehensive test cases, test plans, coverage analysis |
| **xlsx** | Read Excel .xlsx/.xlsm files with sheet selection and CSV export |

## Skill Discovery

Skills are in the `skills/` directory. Each has a `SKILL.md` describing when and how to use it. When a user's task matches a skill, follow the process in its `SKILL.md`.

## Changelog

### v1.6.0
Confluence skill — bug fixes and folder support, all verified against a live Confluence Cloud site.

Fixed:
- **Every printed URL was a dead link.** `_links.webui` is relative to the wiki context path (`/spaces/...`), but it was appended to the bare `base_url`, dropping `/wiki`. Now uses the `base` that ships alongside `webui`, falling back to appending the context path. Affected `--create`, `--get` and `--search`.
- **`--children` always failed** with HTTP 404 — `/children/page` is not a valid v2 endpoint. Now uses `/direct-children`, which also reports each child's type. (`/children`, the obvious alternative, silently omits folders and returns a null type for every row.)
- **Documented invocation could not work.** `go run ${CLAUDE_SKILL_DIR}/confluence.go` fails from any directory outside the module, since Go resolves modules from the working directory. SKILL.md now builds the binary first. It also warns against `go run -C`, which compiles but changes the program's working directory, so a project-local `.claude/confluence-config.yaml` is silently ignored in favour of the global one — publishing to the wrong site rather than failing.

Added:
- `--create-folder` — folders group pages without holding content, and never appear in `--search` results.
- `--parent` on `--update`, to move a page under another page or folder. The v2 API only honours `parentId` on a full update, so the body must be resent.
- `--get`, `--delete` and `--children` now accept a folder ID, falling back to `/api/v2/folders` on 404.

### v1.5.0
- Simplified all 15 SKILL.md files to reduce token usage (~72% reduction, ~1,875 lines removed)
- Removed redundant examples sections that duplicated flags tables
- Eliminated repeated config blocks across Google skills (docs/drive/sheets)
- Condensed verbose PRD templates into structural outlines
- All flags, operations, and config options preserved — only duplicate text removed

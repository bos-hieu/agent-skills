# Agent Skills

A collection of reusable skills for AI coding agents.

## Available Skills

| Skill | Description |
|---|---|
| **browser-test** | Run and debug Playwright browser tests (API, E2E, smoke) |
| **confluence** | Manage Confluence pages — create, read, update, search, delete |
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

### v1.5.0
- Simplified all 15 SKILL.md files to reduce token usage (~72% reduction, ~1,875 lines removed)
- Removed redundant examples sections that duplicated flags tables
- Eliminated repeated config blocks across Google skills (docs/drive/sheets)
- Condensed verbose PRD templates into structural outlines
- All flags, operations, and config options preserved — only duplicate text removed

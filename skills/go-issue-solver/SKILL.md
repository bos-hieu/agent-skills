---
name: go-issue-solver
description: Golang coding tool to analyze and solve any issue in the codebase. Searches relevant files, diagnoses root causes, scaffolds fixes, runs tests, and reports results. Works from an issue description or a failing test/log.
allowed-tools: Bash(go run *), Bash(go test *), Bash(go build *), Bash(go vet *), Bash(grep *), Bash(cat *), Bash(ls *), Bash(find *)
---

Given a bug report, failing test, error log, or feature request, use `${CLAUDE_SKILL_DIR}` to reference the Go file.

## Usage

```bash
go run ${CLAUDE_SKILL_DIR}/issue_solver.go [flags]
```

| Flag | Description |
|---|---|
| `--issue <text>` | Issue description, error message, or log snippet |
| `--file <path>` | Specific file or directory to analyze |
| `--search <term>` | Search codebase for a term (file + line) |
| `--test <pkg>` | Run tests for a package, show failures |
| `--build` | Run `go build ./...`, show errors |
| `--vet` | Run `go vet ./...`, show warnings |
| `--scaffold <type>` | Scaffold boilerplate: `handler`, `service`, `repo`, `migration`, `task` |
| `--name <name>` | Name for scaffolded item (with `--scaffold`) |
| `--fix-imports` | Run goimports on all Go files |
| `--unused` | Find unused exports (deadcode analysis) |
| `--callers <func>` | Find all call sites of a function |

## Scaffold Output Paths

| Type | Path |
|---|---|
| `handler` | `internal/handlers/web/<name>.go` |
| `service` | `internal/services/web/<name>.go` |
| `repo` | `internal/repositories/<name>.go` |
| `migration` | `migrations/<timestamp>_<name>.up.sql` |
| `task` | `internal/tasks/<name>.go` |

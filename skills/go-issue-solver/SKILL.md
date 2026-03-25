---
name: go-issue-solver
description: Golang coding tool to analyze and solve any issue in the codebase. Searches relevant files, diagnoses root causes, scaffolds fixes, runs tests, and reports results. Works from an issue description or a failing test/log.
allowed-tools: Bash(go run *), Bash(go test *), Bash(go build *), Bash(go vet *), Bash(grep *), Bash(cat *), Bash(ls *), Bash(find *)
---

When given a bug report, failing test, error log, or feature request:

1. Parse the issue description to extract key terms, file hints, and error messages.
2. Search the codebase for relevant files using the terms found.
3. Read those files and produce a structured diagnosis.
4. Suggest a concrete fix and optionally scaffold it.
5. Run `go test` on affected packages and report results.
6. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.

## Flags

| Flag | Description |
|---|---|
| `--issue <text>` | Issue description, error message, or log snippet |
| `--file <path>` | Specific file or directory to analyze |
| `--search <term>` | Search the codebase for a term (file + line) |
| `--test <pkg>` | Run tests for a package and show failures |
| `--build` | Run `go build ./...` and show errors |
| `--vet` | Run `go vet ./...` and show warnings |
| `--scaffold <type>` | Scaffold boilerplate: `handler`, `service`, `repo`, `migration`, `task` |
| `--name <name>` | Name for the scaffolded item (used with --scaffold) |
| `--fix-imports` | Run goimports on all Go files |
| `--unused` | Find unused exports (deadcode analysis) |
| `--callers <func>` | Find all call sites of a function name |

## Examples

```bash
# Analyze an issue from error text
go run ${CLAUDE_SKILL_DIR}/issue_solver.go \
  --issue "panic: runtime error: invalid memory address at handlers/web/wallet.go"

# Analyze from a log snippet
go run ${CLAUDE_SKILL_DIR}/issue_solver.go \
  --issue "ERROR: duplicate key value violates unique constraint users_email_key"

# Search codebase for a term
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --search "WalletTransfer"

# Find all callers of a function
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --callers "ProcessDeposit"

# Run tests and show only failures
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --test ./internal/services/...

# Build check
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --build

# Vet check
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --vet

# Scaffold a new handler
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --scaffold handler --name savings_plan

# Scaffold a new migration
go run ${CLAUDE_SKILL_DIR}/issue_solver.go --scaffold migration --name add_index_to_transactions
```

## Scaffold templates

| Type | Output path | What it creates |
|---|---|---|
| `handler` | `internal/handlers/web/<name>.go` | Gin handler with CRUD skeleton |
| `service` | `internal/services/web/<name>.go` | Service interface + struct |
| `repo` | `internal/repositories/<name>.go` | Repository interface + GORM impl |
| `migration` | `migrations/<timestamp>_<name>.up.sql` | SQL migration file |
| `task` | `internal/tasks/<name>.go` | Asynq task definition + handler |

## Diagnosis output format

```
=== Issue Analysis ===
Input: "<your issue text>"

Matched files:
  internal/handlers/web/wallet.go  (line 142: relevant snippet)
  internal/services/web/wallet_service.go

Root cause (likely):
  <explanation>

Suggested fix:
  <concrete steps>

Affected packages:
  ./internal/handlers/web
  ./internal/services/web

Run tests:
  go test -v ./internal/handlers/web/... ./internal/services/web/...
```

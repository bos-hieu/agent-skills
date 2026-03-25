---
name: generating-claude-instructions
description: Generate a CLAUDE.md file at the project root by deeply exploring the actual source code. The file must contain only verified facts — never guess or infer. The goal is to eliminate codebase re-exploration in every future session, saving tokens and time.
---

# Generating Claude Instructions (CLAUDE.md)

## Overview

Generate a CLAUDE.md file at the project root by deeply exploring the actual source code. The file must contain only verified facts — never guess or infer. The goal is to eliminate codebase re-exploration in every future session, saving tokens and time.

## When to Use

- Entering a new repository that has no CLAUDE.md
- User asks to create or regenerate CLAUDE.md
- Existing CLAUDE.md is outdated or shallow

## Process

### Step 1: Launch Parallel Exploration Agents

Dispatch 4 agents simultaneously, each exploring a different layer:

**Agent 1 — Entry Points & Transport Layer:**
- Entry point / main file / bootstrap
- Handler / controller / route files (read 3-4 fully)
- Route registration patterns, middleware chain
- Request binding, response wrapper, error handling patterns
- WebSocket or real-time handlers if present

**Agent 2 — Business Logic Layer:**
- Service / usecase / domain files (read 3-4 fully)
- Constructor patterns, dependency injection
- Data transformation patterns (DTO/proto/entity conversion)
- Cache wrapper patterns if present
- Error handling, retry, observability patterns

**Agent 3 — Shared Packages & Models:**
- Entity / model / type definitions (read several)
- Utility packages, constants, helpers
- Third-party integration wrappers
- Middleware implementations
- Auth, i18n, logging packages
- SDK or protocol packages

**Agent 4 — Infrastructure & Config:**
- Config system (read actual config files)
- Dockerfile / docker-compose
- Makefile / package.json / build scripts
- Port/interface definitions (read 4-5)
- Database layer (ORM, repositories, migrations)
- Test files (to understand testing patterns)
- README, .env.example, CI files

### Step 2: Synthesize into CLAUDE.md

Write the file using ONLY facts verified by agents reading actual source code.

### Required Sections

```markdown
# Project Name

One-line description: what it is, language, framework, purpose.

## Quick Reference
- Module/package name, language version, framework version
- Key dependency versions (from lockfile/module file)
- Database(s) and data stores used
- Repository URL
- Entry point file path
- Main DI/wiring file path
- Private module setup if needed (GOPRIVATE, npm registry, etc.)

## Internal Library Dependencies
List internal/private libraries with versions and what they provide.

## Architecture
Full directory tree with one-line purpose for each directory.
Include ALL directories, not just top-level.

## Handler/Controller Patterns
Real code snippet showing:
- Constructor signature
- Route registration
- Request binding + validation
- Response wrapper usage
- How errors are returned

Note any variant patterns (REST vs WebSocket vs RPC).

## Service/Usecase Patterns
Real code snippet showing:
- Constructor with dependency injection
- Method structure (tracing, gRPC call, transformation, error handling)
- Cache wrapper pattern if present
- Data transformation approach (copier, manual mapping, etc.)

## Context/Request Utilities
List middleware-injected values and extraction functions
(e.g., GetCurrentUser, GetLocale, GetRequestID).

## Middleware Chain
Exact order, numbered. Note which paths are excluded.
Note route-level middleware options.

## Config System
How config is loaded, file format, how to access values.
List environments and their config files.
List backend services/dependencies with connection details format.

## Domain Modules
List ALL domain modules (handlers, usecases, etc.).
Note which have cache wrappers.

## Entity/Model Patterns
Base struct pattern, common fields, builder/options patterns.
Count of entity files.

## Makefile / Build Commands
Only commands developers actually use. No generic fallbacks.

## Docker
Build stages, base images, entrypoint. Keep brief.

## Key Constants
Timeouts, limits, defaults, supported locales, etc.
```

### Rules

1. **Every fact must come from reading actual source files** — not from directory names, README claims, or assumptions
2. **Include real code snippets** — show actual constructor signatures, actual wrapper functions, actual patterns from the codebase
3. **No tutorials** — don't explain what hexagonal architecture is, just document that the project uses it and how
4. **No filler** — every line should save future sessions from having to discover it
5. **Dense and scannable** — use code blocks for patterns, bullet lists for references, tables for mappings
6. **Version-specific** — include actual versions from lockfile, not ranges
7. **List ALL domain modules** — future sessions need to know the full scope without globbing

## Common Mistakes

- Writing generic descriptions instead of project-specific verified facts
- Including `go test ./...` or `npm test` without checking if tests actually exist/pass
- Describing patterns from only one file and assuming all follow it
- Missing variant patterns (e.g., REST handlers vs WebSocket vs RPC handlers)
- Forgetting to document cache wrapper patterns
- Not listing ALL domain modules (forcing future re-exploration)
- Including build commands nobody uses

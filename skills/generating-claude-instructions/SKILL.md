---
name: generating-claude-instructions
description: Generate a CLAUDE.md file at the project root by deeply exploring the actual source code. The file must contain only verified facts — never guess or infer. The goal is to eliminate codebase re-exploration in every future session, saving tokens and time.
---

## When to Use

- New repository with no CLAUDE.md
- User asks to create or regenerate CLAUDE.md
- Existing CLAUDE.md is outdated or shallow

## Process

### Step 1: Launch 4 Parallel Exploration Agents

**Agent 1 — Entry Points & Transport:** Entry point/main file, handler/controller/route files (read 3-4 fully), route registration, middleware chain, request binding, response/error patterns, WebSocket handlers.

**Agent 2 — Business Logic:** Service/usecase/domain files (read 3-4 fully), constructor patterns, DI, data transformation (DTO/proto/entity), cache wrappers, error/retry/observability patterns.

**Agent 3 — Shared Packages & Models:** Entity/model/type definitions, utilities, constants, third-party wrappers, middleware, auth/i18n/logging packages.

**Agent 4 — Infrastructure & Config:** Config system, Dockerfile/docker-compose, Makefile/build scripts, port/interface definitions (read 4-5), DB layer (ORM/repos/migrations), test files, README/.env.example/CI.

### Step 2: Write CLAUDE.md

Use ONLY verified facts from agent reads. Required sections:

```
# Project Name — one-line description

## Quick Reference
Module name, language/framework versions, key deps, DB(s), entry point, DI wiring file, private module setup

## Internal Library Dependencies
List with versions and purpose

## Architecture
Full directory tree with one-line purpose per directory (ALL dirs, not just top-level)

## Handler/Controller Patterns
Real code snippet: constructor, route registration, request binding, validation, response wrapper, error return. Note variant patterns.

## Service/Usecase Patterns
Real code snippet: constructor with DI, method structure, cache wrapper, data transformation approach.

## Context/Request Utilities
Middleware-injected values and extraction functions

## Middleware Chain
Exact order, numbered. Excluded paths. Route-level options.

## Config System
How loaded, file format, access method, environments, backend services

## Domain Modules
ALL modules listed. Note cache wrappers.

## Entity/Model Patterns
Base struct, common fields, builder/options. File count.

## Makefile / Build Commands
Only commands developers actually use.

## Docker
Build stages, base images, entrypoint. Brief.

## Key Constants
Timeouts, limits, defaults, locales.
```

### Rules

1. Every fact must come from reading actual source files
2. Include real code snippets from the codebase
3. No tutorials or generic explanations — document what the project uses and how
4. No filler — every line saves future sessions from re-discovery
5. Include actual versions from lockfile, not ranges
6. List ALL domain modules to avoid future re-exploration

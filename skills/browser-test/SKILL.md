---
name: browser-test
description: Run and debug Playwright browser tests for this project (API, E2E, smoke). Wraps the existing tests/frontend Playwright setup with smart filtering, result parsing, and failure reporting.
allowed-tools: Bash(go run *), Bash(npx *), Bash(node *), Bash(npm *), Bash(cat *), Bash(ls *)
---

When the user wants to run browser tests, debug a failing test, or add a new Playwright test:

1. Use the Go runner to orchestrate Playwright and parse results.
2. Filter by project (api, e2e, smoke), file pattern, or test name.
3. Show clean pass/fail output with failure details.
4. For new tests, scaffold the correct file in the right directory.
5. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.

## Test layout (this project)

```
tests/frontend/
├── playwright.config.js
└── tests/
    ├── api/       *.api.spec.js   — API contract tests (no browser)
    ├── e2e/       *.e2e.spec.js   — Full browser E2E tests
    ├── smoke/     split-deployment.test.js
    ├── unit/      *.test.js       — JS unit tests (node --test)
    └── support/   helpers
```

## Flags

| Flag | Description | Default |
|---|---|---|
| `--project <name>` | Test project: `api`, `e2e`, `smoke` | all |
| `--file <glob>` | Filter test files by pattern | all |
| `--test <name>` | Filter by test name (substring) | all |
| `--headed` | Run browser in headed (visible) mode | headless |
| `--debug` | Run with PWDEBUG=1 (Playwright inspector) | off |
| `--retries <n>` | Override retry count | config default |
| `--base-url <url>` | Override BASE_URL | http://localhost:3030 |
| `--install` | Install Playwright + chromium (`npm install`) | — |
| `--list-tests` | List discovered tests without running | — |
| `--report` | Open HTML report after run | — |

## Examples

```bash
# First-time setup
go run ${CLAUDE_SKILL_DIR}/main.go --install

# Run all API tests
go run ${CLAUDE_SKILL_DIR}/main.go --project api

# Run all E2E tests
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e

# Run a specific test file
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e --file admin-login

# Run tests matching a name
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e --test "login with valid credentials"

# Debug a failing test interactively
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e --file admin-login --debug

# Run against a different server
go run ${CLAUDE_SKILL_DIR}/main.go --project api --base-url http://localhost:3031

# List available tests without running
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e --list-tests

# Run with retries for flaky tests
go run ${CLAUDE_SKILL_DIR}/main.go --project e2e --retries 2
```

## Adding a new test

Scaffold:
- API test → `tests/frontend/tests/api/<name>.api.spec.js`
- E2E test → `tests/frontend/tests/e2e/<name>.e2e.spec.js`

Template:
```js
import { test, expect } from '@playwright/test';

test.describe('<Feature>', () => {
  test('<scenario>', async ({ page, request }) => {
    // API test: use `request`
    // E2E test: use `page`
  });
});
```

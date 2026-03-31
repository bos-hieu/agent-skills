---
name: browser-test
description: Run and debug Playwright browser tests for this project (API, E2E, smoke). Wraps the existing tests/frontend Playwright setup with smart filtering, result parsing, and failure reporting.
allowed-tools: Bash(go run *), Bash(npx *), Bash(node *), Bash(npm *), Bash(cat *), Bash(ls *)
---

When the user wants to run browser tests, debug a failing test, or add a new Playwright test, use `${CLAUDE_SKILL_DIR}` to reference the Go file.

## Test Layout

```
tests/frontend/
├── playwright.config.js
└── tests/
    ├── api/    *.api.spec.js
    ├── e2e/    *.e2e.spec.js
    ├── smoke/  split-deployment.test.js
    ├── unit/   *.test.js (node --test)
    └── support/ helpers
```

## Usage

```bash
go run ${CLAUDE_SKILL_DIR}/main.go [flags]
```

| Flag | Description |
|---|---|
| `--project <name>` | Test project: `api`, `e2e`, `smoke` (default: all) |
| `--file <glob>` | Filter test files by pattern |
| `--test <name>` | Filter by test name substring |
| `--headed` | Run browser in headed mode |
| `--debug` | Run with PWDEBUG=1 |
| `--retries <n>` | Override retry count |
| `--base-url <url>` | Override BASE_URL (default: http://localhost:3030) |
| `--install` | Install Playwright + chromium |
| `--list-tests` | List discovered tests without running |
| `--report` | Open HTML report after run |

## Adding a New Test

Place in `tests/frontend/tests/api/<name>.api.spec.js` or `tests/frontend/tests/e2e/<name>.e2e.spec.js`:

```js
import { test, expect } from '@playwright/test';
test.describe('<Feature>', () => {
  test('<scenario>', async ({ page, request }) => {
    // API test: use `request`; E2E test: use `page`
  });
});
```

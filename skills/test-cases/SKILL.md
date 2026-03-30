---
name: test-cases
description: Generate, organize, and manage comprehensive test cases for features, code changes, or PRs. Covers test case generation, test plan creation, coverage analysis, bug report templates, and test matrices.
---

## Process

### Step 1: Understand What Needs Testing

Read the relevant inputs (PRD, user story, diff, code module) and identify:
- Feature/component under test, all inputs/outputs/side effects
- User roles, external dependencies, acceptance criteria
- Non-functional requirements (performance, security, accessibility)

Determine relevant test types: Unit, Integration, E2E, API, UI, Regression, Smoke, Performance.

### Step 2: Generate Test Cases

Cover ALL relevant categories:

1. **Happy Path** — Standard successful workflows, all acceptance criteria exercised
2. **Edge Cases / Boundaries** — Min/max values, empty/null inputs, boundary transitions, unicode, concurrency, timezones
3. **Error Handling / Negative** — Invalid inputs, missing fields, unauthorized access, network failures, DB constraint violations, rate limiting
4. **Security** — Auth boundaries, input sanitization (XSS/SQLi/command injection), CSRF, sensitive data exposure, session management, file upload restrictions
5. **Performance** — Response time, high concurrency, large datasets, cache behavior, resource cleanup
6. **Cross-Browser / Platform** — Browsers, mobile vs desktop, OS-specific, touch vs mouse (if applicable)
7. **Accessibility** — Keyboard nav, screen reader, color contrast, ARIA, form labels (if applicable)
8. **Data Validation** — Field-level rules, cross-field validation, format validation, business rules, DB integrity

### Step 3: Structure Output

| Column | Description |
|--------|-------------|
| ID | e.g., TC-001 |
| Category | From Step 2 |
| Title | What is being tested |
| Preconditions | Required state/setup |
| Steps | Numbered actions |
| Expected Result | Observable pass/fail outcome |
| Priority | P0 (blocker), P1 (critical), P2 (major), P3 (minor) |
| Type | Manual or Automated |

### Step 4: BDD Format (When Requested)

```gherkin
Feature: [Name]
  Scenario: [Title]
    Given [preconditions]
    When [action]
    Then [expected result]

  Scenario Outline: [Parameterized Title]
    Given [preconditions with <param>]
    When [action with <param>]
    Then [result with <param>]
    Examples:
      | param | expected |
      | val1  | res1     |
```

### Step 5: Test Matrix (When Requested)

List variables (browser, OS, role, input type) and values. Generate pairwise combinations (not full cartesian). Present as table with Config ID, variable columns, and Expected.

### Step 6: Test Plan (When Requested)

Sections: Overview, Scope (in/out), Test Strategy, Entry Criteria, Exit Criteria, Test Environment, Schedule, Risks & Mitigations, Test Cases.

### Step 7: Coverage Analysis (When Requested)

Read existing tests, map to code paths, identify gaps. Present as: `Gap ID | Area | Missing Coverage | Suggested Test | Priority`

### Step 8: Bug Report Template (When Requested)

```
Title, Environment (OS/Browser/Device, App version, Environment),
Severity (Blocker/Critical/Major/Minor/Trivial),
Steps to Reproduce, Expected Result, Actual Result,
Reproducibility (Always/Intermittent/Once), Attachments, Additional Context
```

## Rules

1. Always read relevant code/requirements/PR before generating — never assume
2. Every test case must have a clear, specific expected result
3. Prioritize by user impact — not everything is P0
4. Use concrete values, not placeholders
5. Keep steps atomic and reproducible
6. Flag assumptions explicitly
7. Don't duplicate test cases — merge or parameterize

---
name: test-cases
description: Generate, organize, and manage comprehensive test cases for features, code changes, or PRs. Covers test case generation, test plan creation, coverage analysis, bug report templates, and test matrices.
---

# Test Cases

## Overview

Generate, organize, and manage comprehensive test cases for features, code changes, or PRs. This skill guides you through analyzing what needs testing and producing structured, actionable test cases across multiple categories and test types.

## When to Use

- User asks to generate test cases for a feature, story, or requirement
- User asks to review existing test cases for completeness or gaps
- User wants a test plan for a project, release, or sprint
- User needs a bug report template
- User asks for test coverage analysis on code or a PR
- User wants test cases in BDD (Given/When/Then) format
- User needs a test matrix for combinatorial testing

## Process

### Step 1: Understand What Needs Testing

Gather context before generating anything. Read the relevant inputs thoroughly:

- **Feature/requirement**: Read the PRD, user story, acceptance criteria, or ticket description
- **Code change or PR**: Read the diff, changed files, and surrounding code to understand behavior changes
- **Existing code**: Read the module, its public API, and any existing tests

Identify and document:

1. The feature or component under test
2. All inputs, outputs, and side effects
3. User roles and permissions involved
4. External dependencies (APIs, databases, third-party services)
5. Stated acceptance criteria or business rules
6. Non-functional requirements (performance, security, accessibility)

Ask the user to clarify scope if the request is ambiguous. Determine which test types are relevant:

| Type | When to Include |
|------|----------------|
| Unit | Individual functions, methods, or classes |
| Integration | Interactions between modules, services, or databases |
| E2E | Full user workflows through the system |
| API | REST/GraphQL/gRPC endpoint behavior |
| UI | Visual rendering, layout, user interactions |
| Regression | Changes that could break existing behavior |
| Smoke | Critical-path validation after deployment |
| Performance | Load, stress, response time requirements |

### Step 2: Generate Test Cases

Produce test cases organized by category. Cover ALL of the following categories that are relevant to the feature:

**Category 1 -- Happy Path / Positive Scenarios**
- Standard successful workflows with valid inputs
- All primary use cases functioning as designed
- Each acceptance criterion exercised at least once

**Category 2 -- Edge Cases and Boundary Values**
- Minimum and maximum allowed values
- Empty inputs, zero-length strings, null/undefined
- Boundary transitions (e.g., 0 to 1, max-1 to max, max to max+1)
- Unicode, special characters, extremely long strings
- Concurrent or simultaneous operations
- Time zone and locale variations

**Category 3 -- Error Handling / Negative Scenarios**
- Invalid inputs (wrong type, out of range, malformed)
- Missing required fields
- Unauthorized access attempts
- Network failures, timeouts, service unavailability
- Database constraint violations
- Rate limiting and throttling behavior

**Category 4 -- Security Considerations**
- Authentication and authorization boundaries
- Input sanitization (XSS, SQL injection, command injection)
- CSRF protection
- Sensitive data exposure (PII in logs, responses, URLs)
- Session management (expiry, hijacking, fixation)
- File upload restrictions (type, size, content validation)

**Category 5 -- Performance Considerations**
- Response time under normal load
- Behavior under high concurrency
- Large dataset handling (pagination, streaming, memory)
- Cache behavior (hit, miss, invalidation, expiry)
- Resource cleanup (connections, file handles, memory leaks)

**Category 6 -- Cross-Browser / Cross-Platform** (if applicable)
- Supported browsers and versions
- Mobile vs desktop viewports
- OS-specific behavior
- Touch vs mouse interactions

**Category 7 -- Accessibility** (if applicable)
- Keyboard navigation and focus management
- Screen reader compatibility
- Color contrast and text scaling
- ARIA attributes and semantic HTML
- Form labels and error announcements

**Category 8 -- Data Validation**
- Field-level validation rules
- Cross-field validation (dependent fields)
- Format validation (email, phone, date, URL)
- Business rule validation (e.g., end date after start date)
- Database integrity constraints

### Step 3: Structure the Output

Present test cases in a table with the following columns:

| Column | Description |
|--------|-------------|
| ID | Unique identifier, e.g., TC-001 |
| Category | Which category from Step 2 |
| Test Case Title | Clear, concise description of what is being tested |
| Preconditions | State or setup required before executing |
| Steps | Numbered sequence of actions |
| Expected Result | Observable outcome that confirms pass/fail |
| Priority | P0 (blocker), P1 (critical), P2 (major), P3 (minor) |
| Type | Manual or Automated |

**Priority guidelines:**

- **P0 -- Blocker**: Core functionality that prevents release if broken. Must pass before any deployment.
- **P1 -- Critical**: Important functionality affecting many users. Must pass before release.
- **P2 -- Major**: Secondary functionality or uncommon workflows. Should pass before release.
- **P3 -- Minor**: Edge cases, cosmetic issues, rare scenarios. Nice to have before release.

**Example row:**

| ID | Category | Test Case Title | Preconditions | Steps | Expected Result | Priority | Type |
|----|----------|----------------|---------------|-------|-----------------|----------|------|
| TC-001 | Happy Path | User creates account with valid email | None | 1. Navigate to signup page 2. Enter valid email and password 3. Click Submit | Account is created, confirmation email sent, user redirected to dashboard | P0 | Automated |

### Step 4: BDD Format (When Requested)

If the user requests BDD format, convert each test case to Given/When/Then syntax:

```gherkin
Feature: [Feature Name]

  Scenario: [Test Case Title]
    Given [preconditions]
    When [action performed]
    Then [expected result]
    And [additional expected result]

  Scenario Outline: [Parameterized Test Case Title]
    Given [preconditions with <parameter>]
    When [action with <parameter>]
    Then [expected result with <parameter>]

    Examples:
      | parameter | expected |
      | value1    | result1  |
      | value2    | result2  |
```

Use Scenario Outline with Examples tables for data-driven test cases that share the same steps but differ in input/output values.

### Step 5: Test Matrix (When Requested)

For combinatorial testing, generate a matrix identifying the variables and their values:

1. List all variables (e.g., browser, OS, user role, input type)
2. List possible values for each variable
3. Generate combinations using pairwise coverage (not full cartesian product) to keep the matrix manageable
4. Present as a table where each row is a test configuration

Example:

| Config | Browser | OS | User Role | Expected |
|--------|---------|-----|-----------|----------|
| M-001 | Chrome | Windows | Admin | Pass |
| M-002 | Firefox | macOS | Viewer | Pass |
| M-003 | Safari | iOS | Editor | Pass |

### Step 6: Test Plan (When Requested)

If the user asks for a test plan, produce a document with these sections:

1. **Overview**: What is being tested, release or sprint reference
2. **Scope**: In-scope and out-of-scope items, explicitly stated
3. **Test Strategy**: Types of testing to perform, tools and frameworks to use
4. **Entry Criteria**: Conditions that must be met before testing starts (e.g., build passes, environment ready)
5. **Exit Criteria**: Conditions that must be met before testing is complete (e.g., all P0/P1 pass, coverage threshold met)
6. **Test Environment**: Infrastructure, data, accounts, and configuration needed
7. **Schedule**: Phases with dates or sprint references
8. **Risks and Mitigations**: Known risks to test execution or quality, with mitigation plans
9. **Test Cases**: Reference or embed the generated test cases from Steps 2-3

### Step 7: Coverage Analysis (When Requested)

When reviewing existing test cases or code for coverage gaps:

1. Read all existing test files for the component
2. Map each test to the code path it exercises
3. Identify untested paths:
   - Code branches (if/else/switch) not covered
   - Error handling paths not tested
   - Public API methods without corresponding tests
   - Missing boundary value tests
   - Missing negative/error scenario tests
4. Present findings as a gap report:

| Gap ID | Area | Missing Coverage | Suggested Test Case | Priority |
|--------|------|-----------------|-------------------|----------|
| G-001 | Error handling | No test for DB connection failure | Test service behavior when DB is unreachable | P1 |

### Step 8: Bug Report Template (When Requested)

When the user asks for a bug report template, provide this structure:

```
Title: [Brief, descriptive summary]

Environment:
- OS / Browser / Device:
- App version / Build:
- Environment (staging/production):

Severity: [Blocker / Critical / Major / Minor / Trivial]

Steps to Reproduce:
1. [Step 1]
2. [Step 2]
3. [Step 3]

Expected Result:
[What should happen]

Actual Result:
[What actually happens]

Reproducibility: [Always / Intermittent / Once]

Attachments:
[Screenshots, logs, network traces, video]

Additional Context:
[Related tickets, recent deployments, workarounds]
```

## Rules

1. **Never generate test cases without first reading the relevant code, requirements, or PR** -- test cases must be grounded in actual behavior, not assumptions
2. **Every test case must have a clear expected result** -- vague outcomes like "works correctly" are not acceptable
3. **Assign priorities based on user impact** -- not all test cases are equal; prioritize ruthlessly
4. **Prefer concrete values in examples** -- use realistic test data, not placeholders like "test123"
5. **Keep test case steps atomic and reproducible** -- another person must be able to execute them without asking questions
6. **Flag assumptions explicitly** -- if you inferred a requirement, state it and ask the user to confirm
7. **Do not duplicate test cases** -- if two cases test the same code path with the same logic, merge them or use parameterization
8. **Include both the test type and automation recommendation** -- indicate whether each test is better suited for manual or automated execution and why

## Common Mistakes

- Generating only happy path tests and ignoring error handling and edge cases
- Writing vague steps like "enter invalid data" without specifying what invalid data means
- Skipping security test cases for features that handle user input
- Not considering state dependencies between test cases
- Assigning everything P0 priority instead of making meaningful priority distinctions
- Generating hundreds of low-value test cases instead of fewer high-impact ones
- Forgetting to test the absence of behavior (e.g., deleted user cannot log in)
- Not covering data validation at both client and server layers
- Ignoring cleanup and teardown requirements in preconditions
- Writing test cases that depend on a specific test execution order without documenting that dependency

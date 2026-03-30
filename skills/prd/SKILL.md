---
name: prd
description: Generate professional Product Requirements Documents, user stories, feature specs, competitive analyses, release notes, and stakeholder updates. Guides structured product thinking from problem definition through release planning.
---

## Artifact Types

| Artifact | When to Use |
|---|---|
| **Full PRD** | New product or major feature needing cross-functional alignment |
| **User Stories** | Breaking down a feature into implementable units |
| **Feature Spec** | Smaller feature not warranting a full PRD |
| **Competitive Analysis** | Evaluating market landscape or justifying decisions |
| **Release Notes** | Communicating shipped changes to users |
| **Stakeholder Update** | Reporting progress to leadership |

If ambiguous, ask which artifact before proceeding.

## Process

1. **Identify artifact type** from the table above
2. **Gather context** — ask user for missing critical info rather than inventing it
3. **Generate** using the appropriate structure below

## Full PRD Structure

```
# [Name] - Product Requirements Document
Metadata: Author, Date, Status, Version, Reviewers

1. Problem Statement: What problem, who has it, evidence/data, cost of inaction
2. Goals & Success Metrics: Objective, Key Results table (Metric|Current|Target|Timeline), Non-Goals
3. User Personas: Description, goals, pain points, technical proficiency
4. User Stories: Table (ID|Priority|Story|Acceptance Criteria)
5. Scope: In Scope, Out of Scope (with reasons), Future Considerations
6. Requirements:
   - Functional: Table (ID|Requirement|Priority P0-P3|Notes)
   - Non-Functional: Performance, Security, Scalability, Accessibility, Reliability
   - P0=launch blocker, P1=important with workarounds, P2=deferrable, P3=nice-to-have
7. User Flows: Numbered steps, wireframe descriptions, edge cases & error states
8. Technical Considerations: Dependencies, API requirements, constraints, data needs
9. Release Plan: Phases table (Phase|Scope|Date|Gate), rollout strategy, rollback criteria
10. Risks & Mitigations: Table (Risk|Likelihood|Impact|Mitigation)
11. Open Questions: Table (Question|Owner|Due|Resolution)
12. Appendix: Links to research, designs, glossary
```

## User Stories Format

Summary table first: `ID | Story | Priority | Points | Dependencies`

Each story:
```
## [US-XXX] [Title]
Priority, Story Points, Sprint/Milestone, Dependencies

As a [persona], I want [action], so that [benefit].

Acceptance Criteria (Given/When/Then scenarios for happy path, alternatives, edge cases)
Technical Notes (optional), Out of Scope (optional)
```

## Feature Spec Structure

```
# Feature Spec: [Name]
Metadata: Author, Date, Status, Parent PRD

Problem (2-3 sentences), Proposed Solution, User Stories (3-7),
Requirements table, Design notes, Technical Approach, Success Metrics, Risks, Open Questions
```

## Competitive Analysis Structure

```
# Competitive Analysis: [Area]
Date, Author, Purpose

Market Overview, Feature Comparison table, Competitor Deep Dives (Strengths/Weaknesses/Differentiator/Pricing),
Gaps & Opportunities, Recommendations, Sources
```

## Release Notes Structure

```
# Release Notes - [Product] [Version]
Release Date, Highlights, New Features, Improvements, Bug Fixes, Breaking Changes, Known Issues, Deprecations
```

When generating from commits/PRs: group by type, rewrite to user-facing language, omit internal refactors/CI changes, highlight breaking changes.

## Stakeholder Update Structure

```
# [Project] - Status Update
Period, Author, Overall Status (On Track/At Risk/Blocked)

Summary, Key Accomplishments, Metrics table, Risks & Blockers, Upcoming Milestones, Decisions Needed, Next Steps
```

## Writing Rules

1. Start with Why — problem before solution
2. Measurable success criteria are mandatory
3. Separate must-have (P0) from nice-to-have (P3) — if everything is P0, nothing is prioritized
4. Name risks with mitigations
5. Use plain language for the broadest audience
6. Reference real data; flag gaps as risks
7. Be explicit about scope boundaries
8. Ask for context before generating — never invent data

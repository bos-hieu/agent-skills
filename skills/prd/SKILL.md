---
name: prd
description: Generate professional Product Requirements Documents, user stories, feature specs, competitive analyses, release notes, and stakeholder updates. Guides structured product thinking from problem definition through release planning.
---

# PRD & Product Artifacts

## Overview

Generate professional product management documents including full PRDs, user stories, feature specs, competitive analysis templates, release notes, and stakeholder updates. Every artifact starts with "Why" before "What" and includes measurable success criteria.

## When to Use

- User asks to create a PRD, product requirements document, or product spec
- User needs user stories or acceptance criteria written
- User asks for a feature spec or lightweight requirements document
- User wants to generate a competitive analysis
- User needs release notes drafted from commits, PRs, or changelogs
- User asks for a stakeholder update, status report, or executive summary
- User wants to structure product thinking around a new feature or initiative

## Process

### Step 1: Identify the Artifact Type

Determine which artifact the user needs:

| Artifact | When to Use |
|---|---|
| **Full PRD** | New product or major feature requiring cross-functional alignment |
| **User Stories** | Breaking down a feature into implementable units of work |
| **Feature Spec** | Smaller feature that does not warrant a full PRD |
| **Competitive Analysis** | Evaluating market landscape or justifying product decisions |
| **Release Notes** | Communicating shipped changes to users |
| **Stakeholder Update** | Reporting progress to leadership or cross-functional partners |

If the user's request is ambiguous, ask which artifact they need before proceeding.

### Step 2: Gather Context

Before writing any artifact, collect the necessary inputs. Ask the user for any missing critical information rather than inventing it.

**For PRDs and Feature Specs:**
- What problem are we solving? Who has this problem?
- What evidence or data supports this problem? (user research, support tickets, metrics)
- Who are the target users?
- What does success look like? How will we measure it?
- Are there known constraints? (timeline, technical, regulatory)
- What has already been tried or considered?

**For User Stories:**
- Which feature or PRD do these stories belong to?
- Who are the user personas involved?
- What are the acceptance criteria expectations?

**For Competitive Analysis:**
- Which competitors to include?
- Which features or dimensions to compare?
- What decision is this analysis informing?

**For Release Notes:**
- What commits, PRs, or changelog entries to summarize?
- Who is the audience? (end users, developers, internal stakeholders)
- What version or release name?

**For Stakeholder Updates:**
- What time period does this cover?
- Who is the audience? (executive, cross-functional team, board)
- What are the key milestones, blockers, and decisions needed?

### Step 3: Generate the Artifact

Use the appropriate template below. Adapt section depth to the scope of the project -- a small feature does not need the same detail as a platform initiative.

---

## Artifact Templates

### Full PRD

```markdown
# [Product/Feature Name] - Product Requirements Document

| Field | Value |
|---|---|
| Author | [Name] |
| Date | [Date] |
| Status | Draft / In Review / Approved |
| Version | 1.0 |
| Last Updated | [Date] |
| Reviewers | [Names] |

## 1. Problem Statement

### What problem are we solving?
[Describe the problem in plain language. Focus on the user's pain, not the solution.]

### Who has this problem?
[Identify affected user segments and estimate scale.]

### Evidence & Data
[Reference user research, support tickets, analytics, market data, or interviews that validate this problem. Include specific numbers when available.]

### What happens if we do nothing?
[Describe the cost of inaction -- churn, lost revenue, operational burden, competitive risk.]

## 2. Goals & Success Metrics

### Objective
[One clear sentence describing what success looks like.]

### Key Results
| Metric | Current | Target | Timeline |
|---|---|---|---|
| [KPI 1] | [Baseline] | [Goal] | [When] |
| [KPI 2] | [Baseline] | [Goal] | [When] |
| [KPI 3] | [Baseline] | [Goal] | [When] |

### Non-Goals
[Explicitly state what this initiative is NOT trying to achieve.]

## 3. User Personas

### Persona 1: [Name/Role]
- **Description:** [Who they are]
- **Goals:** [What they want to accomplish]
- **Pain Points:** [Current frustrations relevant to this problem]
- **Technical Proficiency:** [Low / Medium / High]

[Repeat for each persona]

## 4. User Stories

| ID | Priority | User Story | Acceptance Criteria |
|---|---|---|---|
| US-001 | P0 | As a [persona], I want [action], so that [benefit]. | [Criteria] |
| US-002 | P1 | As a [persona], I want [action], so that [benefit]. | [Criteria] |

## 5. Scope

### In Scope
- [Feature/capability 1]
- [Feature/capability 2]

### Out of Scope
- [Explicitly excluded item 1 and why]
- [Explicitly excluded item 2 and why]

### Future Considerations
- [Item that may be addressed in a later phase]

## 6. Requirements

### Functional Requirements

| ID | Requirement | Priority | Notes |
|---|---|---|---|
| FR-001 | [Requirement description] | P0 - Must Have | |
| FR-002 | [Requirement description] | P1 - Should Have | |
| FR-003 | [Requirement description] | P2 - Could Have | |
| FR-004 | [Requirement description] | P3 - Nice to Have | |

**Priority Definitions:**
- **P0 - Must Have:** Launch blocker. Feature cannot ship without this.
- **P1 - Should Have:** Important for target experience but has workarounds.
- **P2 - Could Have:** Desired but can be deferred to a fast-follow.
- **P3 - Nice to Have:** Enhances experience but low impact if absent.

### Non-Functional Requirements

| Category | Requirement |
|---|---|
| Performance | [e.g., Page load under 2s at p95] |
| Security | [e.g., All PII encrypted at rest, SOC 2 compliance] |
| Scalability | [e.g., Support 10x current load without architecture changes] |
| Accessibility | [e.g., WCAG 2.1 AA compliance] |
| Reliability | [e.g., 99.9% uptime SLA] |
| Observability | [e.g., Latency, error rate, and throughput dashboards] |

## 7. User Flows

### Flow 1: [Primary Flow Name]
1. User [action]
2. System [response]
3. User [action]
4. System [response]
5. [End state]

**Wireframe Description:** [Describe the key screens, layouts, and interactions. Reference existing design system components where applicable.]

[Repeat for each major flow]

### Edge Cases & Error States
- [Edge case 1]: [How it should be handled]
- [Error state 1]: [Error message and recovery path]

## 8. Technical Considerations

### Dependencies
- [System/service 1]: [What is needed and current status]
- [System/service 2]: [What is needed and current status]

### API Requirements
- [Endpoint or integration 1]: [Description]
- [Endpoint or integration 2]: [Description]

### Constraints
- [Technical constraint 1]
- [Regulatory constraint 1]

### Data Requirements
- [Data needed, sources, storage, retention]

## 9. Release Plan

### Phases
| Phase | Scope | Target Date | Success Gate |
|---|---|---|---|
| Alpha | [Core flow, internal only] | [Date] | [Gate criteria] |
| Beta | [Expanded scope, limited users] | [Date] | [Gate criteria] |
| GA | [Full launch] | [Date] | [Gate criteria] |

### Rollout Strategy
- [Rollout approach: feature flag, percentage rollout, geographic, etc.]
- [Rollback criteria and process]

### Dependencies & Sequencing
- [What must happen before launch]

## 10. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| [Risk 1] | High/Med/Low | High/Med/Low | [Mitigation strategy] |
| [Risk 2] | High/Med/Low | High/Med/Low | [Mitigation strategy] |

## 11. Open Questions

| # | Question | Owner | Due Date | Resolution |
|---|---|---|---|---|
| 1 | [Question] | [Name] | [Date] | [Pending/Resolved: answer] |

## 12. Appendix

- [Links to research, designs, technical docs, related PRDs]
- [Glossary of domain-specific terms if needed]
```

---

### User Stories

Generate each user story in this format:

```markdown
## [US-XXX] [Short Title]

**Priority:** P0 / P1 / P2 / P3
**Story Points:** [Estimate]
**Sprint/Milestone:** [Target]
**Dependencies:** [List or None]

### User Story
As a [specific persona],
I want [concrete action],
So that [measurable or observable benefit].

### Acceptance Criteria

**Scenario 1: [Happy path name]**
- Given [precondition]
- When [action]
- Then [expected result]

**Scenario 2: [Alternative path name]**
- Given [precondition]
- When [action]
- Then [expected result]

**Scenario 3: [Error/edge case name]**
- Given [precondition]
- When [action]
- Then [expected result]

### Technical Notes
[Implementation hints, API contracts, or architectural notes relevant to this story. Optional.]

### Out of Scope
[Anything that might be assumed in scope but is not. Optional.]
```

When generating multiple stories for a feature, present a summary table first:

```markdown
| ID | Story | Priority | Points | Dependencies |
|---|---|---|---|---|
| US-001 | [Title] | P0 | [Est] | None |
| US-002 | [Title] | P1 | [Est] | US-001 |
```

---

### Feature Spec

A lighter-weight document for smaller features that do not require full PRD rigor.

```markdown
# Feature Spec: [Feature Name]

| Field | Value |
|---|---|
| Author | [Name] |
| Date | [Date] |
| Status | Draft / Approved |
| Parent PRD | [Link or N/A] |

## Problem
[2-3 sentences. What problem does this solve and for whom?]

## Proposed Solution
[Describe the solution. Keep it concise.]

## User Stories
[List 3-7 user stories with acceptance criteria]

## Requirements
| Requirement | Priority | Notes |
|---|---|---|
| [Req 1] | P0 | |
| [Req 2] | P1 | |

## Design
[Describe key screens or interactions. Link to mockups if available.]

## Technical Approach
[High-level implementation approach, key decisions, dependencies.]

## Success Metrics
| Metric | Target |
|---|---|
| [Metric 1] | [Value] |

## Risks
- [Risk 1]: [Mitigation]

## Open Questions
- [Question 1]
```

---

### Competitive Analysis

```markdown
# Competitive Analysis: [Category/Feature Area]

**Date:** [Date]
**Author:** [Name]
**Purpose:** [What decision this analysis informs]

## Market Overview
[Brief landscape summary: market size, trends, key players.]

## Feature Comparison

| Feature | Our Product | [Competitor 1] | [Competitor 2] | [Competitor 3] |
|---|---|---|---|---|
| [Feature 1] | [Status/Rating] | [Status/Rating] | [Status/Rating] | [Status/Rating] |
| [Feature 2] | [Status/Rating] | [Status/Rating] | [Status/Rating] | [Status/Rating] |
| Pricing | [Details] | [Details] | [Details] | [Details] |
| Target Audience | [Details] | [Details] | [Details] | [Details] |

**Legend:** Full Support / Partial / Planned / Not Available

## Competitor Deep Dives

### [Competitor 1]
- **Strengths:** [List]
- **Weaknesses:** [List]
- **Key Differentiator:** [What they do best]
- **Pricing Model:** [Details]
- **Notable Customers:** [If known]

[Repeat for each competitor]

## Gaps & Opportunities
- [Opportunity 1]: [Why it matters, estimated impact]
- [Opportunity 2]: [Why it matters, estimated impact]

## Recommendations
1. [Recommendation with rationale]
2. [Recommendation with rationale]

## Sources
- [Source 1]
- [Source 2]
```

---

### Release Notes

```markdown
# Release Notes - [Product Name] [Version]

**Release Date:** [Date]

## Highlights
[1-2 sentence summary of the most important change in this release.]

## New Features
- **[Feature Name]:** [User-facing description of what it does and why it matters. No implementation details.]

## Improvements
- **[Improvement Name]:** [What changed and how it benefits the user.]

## Bug Fixes
- Fixed an issue where [user-visible symptom]. [Context if helpful.]

## Breaking Changes
- [Description of what changed, who is affected, and migration steps.]

## Known Issues
- [Issue description and workaround if available.]

## Deprecations
- [What is deprecated, when it will be removed, and what to use instead.]
```

When generating release notes from commits or PRs:
1. Group changes by type (feature, improvement, fix, breaking change).
2. Rewrite technical commit messages into user-facing language.
3. Omit internal refactors, dependency bumps, and CI changes unless they affect users.
4. Highlight breaking changes prominently.

---

### Stakeholder Update

```markdown
# [Project/Product Name] - Status Update

**Period:** [Date range]
**Author:** [Name]
**Overall Status:** On Track / At Risk / Blocked

## Summary
[2-3 sentences: what happened, where we are, what is next.]

## Key Accomplishments
- [Accomplishment 1]
- [Accomplishment 2]

## Metrics
| Metric | Previous | Current | Target | Trend |
|---|---|---|---|---|
| [Metric 1] | [Value] | [Value] | [Value] | Up/Down/Flat |

## Risks & Blockers
| Item | Status | Owner | Action Needed |
|---|---|---|---|
| [Risk/Blocker 1] | [Status] | [Name] | [What is needed] |

## Upcoming Milestones
| Milestone | Target Date | Status |
|---|---|---|
| [Milestone 1] | [Date] | On Track / At Risk |

## Decisions Needed
- [Decision 1]: [Context and options. Who needs to decide by when.]

## Next Steps
- [Action 1] - [Owner] - [Due date]
- [Action 2] - [Owner] - [Due date]
```

---

## Writing Guidelines

Apply these principles across all artifacts:

1. **Start with Why.** Every document opens with the problem, not the solution. If the problem is not clear, the solution will not be either.
2. **Measurable success criteria are mandatory.** If you cannot measure it, you cannot know if you succeeded. Push for specific numbers and timelines.
3. **Separate must-have from nice-to-have.** Use the P0-P3 priority framework consistently. A PRD where everything is P0 is a PRD with no priorities.
4. **Name the risks.** Every project has risks. Listing them with mitigations builds credibility and prepares the team.
5. **Use plain language.** Write for the broadest audience that will read the document. Avoid jargon unless the audience is exclusively technical.
6. **Reference real data.** Cite user research, analytics, support tickets, or market data. If no data exists, call that out as a risk and recommend gathering it.
7. **Be explicit about scope boundaries.** What is out of scope is as important as what is in scope. Ambiguity here causes scope creep.
8. **Keep documents alive.** Include status, version, and last-updated fields so readers know if the document is current.

## Common Mistakes

- Writing a solution before clearly defining the problem
- Listing requirements without priority levels, making everything seem equally important
- Omitting success metrics or using vague metrics like "improve user experience"
- Forgetting to state what is out of scope, leading to scope creep
- Writing user stories that describe implementation instead of user value
- Including technical jargon in documents intended for non-technical stakeholders
- Skipping the risks section, leaving the team unprepared for predictable problems
- Generating release notes that read like commit logs instead of user-facing communication
- Not asking the user for context before generating -- filling in placeholders with invented data

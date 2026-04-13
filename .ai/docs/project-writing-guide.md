# Project Writing Guide

Guide for writing `project.md` in the repository root.

Avoid bold Markdown in project documents. It adds no semantic value for AI agents but increases token count. Use headers and lists instead.

## Document Structure

### Header

```markdown
# Title

- Status: Pending | Active | Blocked | Completed
- Started: YYYY-MM-DD

## Overview

Brief description: what, why, and how it fits into the larger system.
```

### Required Sections

- Overview - What this project accomplishes and why
- Goals - 3-7 specific, measurable outcomes
- Scope - In scope and out of scope boundaries
- Success Criteria - Unambiguous, testable outcomes as task list checkboxes
- Deliverables - Concrete outputs (files, code, DRs)

### Optional Sections

- Current State - Existing codebase context
- Technical Approach - High-level implementation strategy (pseudocode OK, no real code)
- Decision Points - Unresolved questions with lettered options (e.g. "1. A/B/C")
- Testing Strategy - How deliverables will be validated
- Decisions - Record of resolved decisions with rationale

## Guidelines

Keep projects focused: clear boundaries, single area of concern, achievable scope. Define what and why, not step-by-step how.

Make success criteria testable:

- Good: "Exit codes are consistent and documented"
- Bad: "Make good error handling"

Do not include implementation source code, resolved design decisions (those belong in DRs), or step-by-step instructions.

## Example

```markdown
# User Authentication

- Status: Active
- Started: 2025-12-01

## Overview

Add user authentication to the application. This enables secure access
and personalised features for registered users.

## Goals

1. Implement user registration and login
2. Add session management
3. Secure API endpoints

## Scope

In Scope:

- Registration, login, session management
- API endpoint authentication

Out of Scope:

- OAuth providers (future work)
- Role-based access control

## Success Criteria

- [ ] Users can register and log in
- [ ] Sessions persist across browser restarts
- [ ] Unauthenticated API requests return 401

## Deliverables

- `internal/auth/` - Authentication package
- `docs/auth.md` - Authentication documentation
```

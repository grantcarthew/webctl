# Project Documents

This directory contains project documents. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

| Project | Title | Status | Started | Completed |
|---------|-------|--------|---------|-----------|
| P-001 | Project Initialization | Completed | 2025-12-11 | 2025-12-11 |
| P-002 | Project Definition | Completed | 2025-12-11 | 2025-12-11 |
| P-003 | CDP Core Library | Completed | 2025-12-11 | 2025-12-15 |
| P-004 | Browser Launch | Completed | 2025-12-11 | 2025-12-15 |
| P-005 | Daemon & IPC | Completed | 2025-12-11 | 2025-12-15 |
| P-006 | CLI Framework | Completed | 2025-12-15 | 2025-12-15 |
| P-007 | Observation Commands | Completed | 2025-12-15 | 2025-12-23 |
| P-008 | Navigation & Interaction Commands | Completed | 2025-12-19 | 2025-12-23 |
| P-009 | Design Review & Validation of P-008 | Completed | 2025-12-23 | 2025-12-24 |
| P-010 | Ready Command Extensions | Completed | 2025-12-23 | 2025-12-25 |
| P-011 | CDP Navigation Debugging | Completed | 2025-12-19 | 2025-12-23 |
| P-012 | Text Output Format | Completed | 2025-12-24 | 2025-12-24 |
| P-013 | Find Command | Completed | 2025-12-25 | 2025-12-25 |
| P-014 | Terminal Colors | Completed | 2025-12-24 | 2025-12-24 |
| P-015 | HTML Formatting for Find and HTML Commands | Completed | 2025-12-26 | 2025-12-26 |
| P-016 | CLI Serve Command | Completed | 2025-12-30 | 2025-12-30 |
| P-017 | CLI CSS Commands | Completed | 2025-12-26 | 2025-12-28 |
| P-018 | Browser Connection Failure Handling | Completed | 2025-12-27 | 2025-12-27 |
| P-019 | Observation Commands Interface Redesign | Completed | 2025-12-28 | 2025-12-28 |
| P-020 | HTML Command Implementation | Completed | 2025-12-28 | 2025-12-28 |
| P-021 | CSS Command Implementation | Completed | 2025-12-28 | 2025-12-28 |
| P-022 | Console Command Implementation | Completed | 2025-12-29 | 2025-12-29 |
| P-023 | Network Command Implementation | Completed | 2025-12-30 | 2025-12-30 |
| P-024 | Cookies Command Implementation | Completed | 2025-12-29 | 2025-12-30 |
| P-025 | Interactive Test Suite | Completed | 2025-12-31 | 2026-01-03 |
| P-026 | Testing start Command | Completed | 2025-12-31 | 2026-01-04 |
| P-027 | Testing stop Command | Completed | 2025-12-31 | 2026-01-06 |
| P-028 | Testing navigate Command | Completed | 2026-01-06 | 2026-01-06 |
| P-029 | Testing serve Command | Completed | 2025-12-31 | 2026-01-06 |
| P-030 | Testing status Command | Completed | 2025-12-31 | 2026-01-07 |
| P-031 | Testing reload Command | Completed | 2025-12-31 | 2026-01-07 |
| P-032 | Testing back Command | Completed | 2025-12-31 | 2026-01-07 |
| P-033 | Testing forward Command | Completed | 2025-12-31 | 2026-01-07 |

Note: Completed projects are in `completed/`

---

## Status Values

- **Proposed** - Project defined, not yet started
- **In Progress** - Currently being worked on
- **Completed** - All success criteria met, deliverables created (move to `completed/`)
- **Blocked** - Waiting on external dependency or decision

---

## Projects vs Design Records

**Projects** are work packages that define **what to build** and **how to validate it**.

**Design Records (DRs)** document **why we chose** a specific approach and the trade-offs.

A single project may generate multiple DRs. Projects describe the work; DRs document the decisions made during that work.

See [p-writing-guide.md](./p-writing-guide.md) for detailed guidance.

---

## Contributing

When creating a new project:

1. List directory to find next number: `ls docs/projects/p-*.md`
2. Use format: `p-<NNN>-<category>-<title>.md`
3. Follow the structure in [p-writing-guide.md](./p-writing-guide.md)
4. Define clear, measurable success criteria
5. Update this README with project entry
6. Link dependencies to other projects

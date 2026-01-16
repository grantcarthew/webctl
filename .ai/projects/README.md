# Project Documents

This directory contains project documents. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

| Project | Title | Status | Started | Completed |
|---------|-------|--------|---------|-----------|
| p-001 | Project Initialization | Completed | 2025-12-11 | 2025-12-11 |
| p-002 | Project Definition | Completed | 2025-12-11 | 2025-12-11 |
| p-003 | CDP Core Library | Completed | 2025-12-11 | 2025-12-15 |
| p-004 | Browser Launch | Completed | 2025-12-11 | 2025-12-15 |
| p-005 | Daemon & IPC | Completed | 2025-12-11 | 2025-12-15 |
| p-006 | CLI Framework | Completed | 2025-12-15 | 2025-12-15 |
| p-007 | Observation Commands | Completed | 2025-12-15 | 2025-12-23 |
| p-008 | Navigation & Interaction Commands | Completed | 2025-12-19 | 2025-12-23 |
| p-009 | Design Review & Validation of p-008 | Completed | 2025-12-23 | 2025-12-24 |
| p-010 | Ready Command Extensions | Completed | 2025-12-23 | 2025-12-25 |
| p-011 | CDP Navigation Debugging | Completed | 2025-12-19 | 2025-12-23 |
| p-012 | Text Output Format | Completed | 2025-12-24 | 2025-12-24 |
| p-013 | Find Command | Completed | 2025-12-25 | 2025-12-25 |
| p-014 | Terminal Colors | Completed | 2025-12-24 | 2025-12-24 |
| p-015 | HTML Formatting for Find and HTML Commands | Completed | 2025-12-26 | 2025-12-26 |
| p-016 | CLI Serve Command | Completed | 2025-12-30 | 2025-12-30 |
| p-017 | CLI CSS Commands | Completed | 2025-12-26 | 2025-12-28 |
| p-018 | Browser Connection Failure Handling | Completed | 2025-12-27 | 2025-12-27 |
| p-019 | Observation Commands Interface Redesign | Completed | 2025-12-28 | 2025-12-28 |
| p-020 | HTML Command Implementation | Completed | 2025-12-28 | 2025-12-28 |
| p-021 | CSS Command Implementation | Completed | 2025-12-28 | 2025-12-28 |
| p-022 | Console Command Implementation | Completed | 2025-12-29 | 2025-12-29 |
| p-023 | Network Command Implementation | Completed | 2025-12-30 | 2025-12-30 |
| p-024 | Cookies Command Implementation | Completed | 2025-12-29 | 2025-12-30 |
| p-025 | Interactive Test Suite | Completed | 2025-12-31 | 2026-01-03 |
| p-026 | Testing start Command | Completed | 2025-12-31 | 2026-01-04 |
| p-027 | Testing stop Command | Completed | 2025-12-31 | 2026-01-06 |
| p-028 | Testing navigate Command | Completed | 2026-01-06 | 2026-01-06 |
| p-029 | Testing serve Command | Completed | 2025-12-31 | 2026-01-06 |
| p-030 | Testing status Command | Completed | 2025-12-31 | 2026-01-07 |
| p-031 | Testing reload Command | Completed | 2025-12-31 | 2026-01-07 |
| p-032 | Testing back Command | Completed | 2025-12-31 | 2026-01-07 |
| p-033 | Testing forward Command | Completed | 2025-12-31 | 2026-01-07 |
| p-034 | Testing html Command | Completed | 2025-12-31 | 2026-01-12 |
| p-035 | Testing css Command | Completed | 2025-12-31 | 2026-01-13 |
| p-036 | Testing console Command | Completed | 2025-12-31 | 2026-01-14 |
| p-037 | Testing network Command | Completed | 2025-12-31 | 2026-01-15 |
| p-038 | Testing cookies Command | In Progress | 2025-12-31 | |
| p-039 | Testing screenshot Command | Proposed | | |
| p-040 | Testing click Command | Proposed | | |
| p-041 | Testing type Command | Proposed | | |
| p-042 | Testing select Command | Proposed | | |
| p-043 | Testing scroll Command | Proposed | | |
| p-044 | Testing focus Command | Proposed | | |
| p-045 | Testing key Command | Proposed | | |
| p-046 | Testing eval Command | Proposed | | |
| p-047 | Testing ready Command | Proposed | | |
| p-048 | Testing clear Command | Proposed | | |
| p-049 | Testing find Command | Proposed | | |
| p-050 | Testing target Command | Proposed | | |
| p-051 | Observation Commands Output Refactor | Completed | 2026-01-08 | 2026-01-09 |
| p-052 | CSS Command Redesign | Completed | 2026-01-12 | 2026-01-12 |
| p-053 | CSS Element Identification | Proposed | | |
| p-054 | Force Stop and Cleanup | Completed | 2026-01-15 | 2026-01-15 |
| p-055 | Test Framework Bash Modules | Completed | 2026-01-15 | 2026-01-15 |
| p-056 | Test Library | Completed | 2026-01-16 | 2026-01-16 |
| p-057 | Test Runner | Completed | 2026-01-16 | 2026-01-16 |
| p-058 | Test Pages | Completed | 2026-01-16 | 2026-01-16 |
| p-059 | CLI Start/Stop Tests | Pending | | |

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

1. List directory to find next number: `ls .ai/projects/p-*.md`
2. Use format: `p-<NNN>-<category>-<title>.md`
3. Follow the structure in [p-writing-guide.md](./p-writing-guide.md)
4. Define clear, measurable success criteria
5. Update this README with project entry
6. Link dependencies to other projects

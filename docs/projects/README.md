# Project Documents

This directory contains project documents. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

| Project | Title | Status | Started | Completed |
|---------|-------|--------|---------|-----------|
| P-001 | Project Initialization | Completed | 2025-12-11 | 2025-12-11 |
| P-002 | Project Definition | Proposed | - | - |
| P-007 | Observation Commands | In Progress | 2025-12-15 | - |
| P-008 | Navigation & Interaction Commands | In Progress | 2025-12-19 | - |
| P-009 | Wait-For Commands | Proposed | - | - |
| P-010 | Polish & Release | Proposed | - | - |
| P-011 | CDP Navigation Debugging | In Progress | 2025-12-19 | - |
| P-012 | Text Output Format | Proposed | - | - |
| P-013 | Find Command | Proposed | - | - |

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

# Context Directory

Project-specific context directory for webctl development.

## Purpose

This directory stores:
- **Research documents** (.md files) - In-depth research findings for design decisions
- **Cloned repositories** (subdirectories) - Temporary clones for documentation review

## Structure

```
.ai/context/
├── README.md              # This file
├── .gitignore             # Ignores cloned repos, tracks .md files
├── *.md                   # Research documents (tracked in git)
└── [cloned-repo]/         # Temporary documentation clones (gitignored)
```

## Usage

### Research Documents

When making significant design decisions, conduct deep research and document findings:

```bash
# Example: CDP timeout research
cd .ai/context
# ... conduct research ...
# Create research document
nvim cdp-timeout-research.md
```

Research documents are tracked in git and serve as references for code comments.

### Cloning Documentation

Clone relevant documentation repositories temporarily for review:

```bash
cd .ai/context
git clone --depth=1 https://github.com/ChromeDevTools/devtools-protocol.git cdp-protocol
```

These clones are automatically gitignored and won't be committed to the repository.

## .gitignore Pattern

The `.gitignore` in this directory uses the pattern:

```gitignore
# Ignore subdirectories (cloned repos)
*/

# Include specific files (like research documents)
!*.md
```

This ensures:
- ✅ Research documents (.md) are tracked
- ✅ Cloned repositories are ignored
- ✅ No accidental commits of large documentation repos

## Examples

See existing research documents:
- `cdp-timeout-research.md` - Deep research on CDP timeout handling limitations

## When to Use

Create research documents when:
- Design decisions need detailed technical justification
- Code comments reference specific research findings
- Complex architectural limitations need comprehensive documentation
- Future maintainers will benefit from understanding "why" decisions were made

## Related

- Main documentation: `.ai/design/design-records/`
- Global AI context: `~/.ai/context/` (see environment.md)

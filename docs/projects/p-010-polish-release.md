# P-010: Polish & Release

- Status: Proposed
- Started: -

## Overview

Final polish, testing, documentation, and release preparation for webctl v1.0.0.

## Goals

1. Comprehensive error handling and edge cases
2. Cross-platform testing
3. Documentation
4. CI/CD pipeline
5. Release binaries

## Scope

In Scope:

- Error message improvements
- Edge case handling
- README and usage documentation
- Man page (optional)
- Cross-platform testing (macOS, Linux)
- CI/CD with GitHub Actions
- Release automation with goreleaser
- Performance testing (10k buffers)

Out of Scope:

- New features
- Windows support (defer to v1.1)
- Remote access features

## Success Criteria

- [ ] All commands have clear error messages
- [ ] Works on macOS (Intel and ARM)
- [ ] Works on Linux (Ubuntu, common distros)
- [ ] README documents all commands
- [ ] CI runs tests on push
- [ ] Release creates binaries for all platforms
- [ ] 10,000 console entries don't cause issues
- [ ] 10,000 network entries with bodies handled gracefully

## Deliverables

- Updated README.md
- `docs/` user documentation
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yml`
- Performance benchmarks

## Technical Design

### Error Message Audit

Review all error paths:
- Element not found → clear selector in message
- Daemon not running → suggest `webctl start`
- Connection refused → check if Chrome crashed
- Timeout → show what was being waited for

### Cross-Platform Testing

| Platform | Test Environment |
|----------|------------------|
| macOS Intel | CI or local |
| macOS ARM | Local (M1/M2) |
| Linux x86_64 | CI (Ubuntu) |
| Linux ARM64 | CI or manual |

### CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./...
      - run: go build ./cmd/webctl
```

### Release Automation

```yaml
# .goreleaser.yml
builds:
  - main: ./cmd/webctl
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
```

### Documentation Structure

```
README.md           # Quick start, installation
docs/
├── installation.md # Detailed install instructions
├── commands.md     # Full command reference
├── examples.md     # Usage examples
└── troubleshooting.md
```

### Performance Testing

Test with realistic load:
1. Navigate to complex page (many console logs)
2. Trigger 10,000 console.log calls
3. Verify buffer works correctly
4. Check memory usage

Same for network:
1. Page that makes many requests
2. Fill network buffer to 10,000
3. Verify body retrieval works
4. Check memory usage

### Known Limitations Documentation

Document clearly:
- Main frame only (no iframe support in v1)
- Native `<select>` only for select command
- Polling-based wait-for (not instant)
- Chrome/Chromium only (no Firefox)

## Dependencies

- P-009 (Wait-For)
- All previous projects complete

## Testing Strategy

Full integration test suite:
1. Start daemon
2. Navigate to test page
3. Run all commands
4. Verify outputs
5. Stop daemon

## Release Checklist

- [ ] All tests pass
- [ ] README up to date
- [ ] CHANGELOG updated
- [ ] Version tagged
- [ ] Binaries built and tested
- [ ] GitHub release published

## Notes

This is the final push to v1.0.0. Focus on stability and documentation over new features. Any feature requests discovered during this phase go to v1.1 backlog.

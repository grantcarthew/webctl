# P-002: Project Definition

- Status: Proposed
- Started: -

## Overview

Before building webctl, we need to validate core assumptions and map out the work ahead. This project investigates whether the fundamental premise works (capturing console/network via CDP), explores the design space, and produces a concrete project roadmap.

This is a "project to define the project" - research, validate, and plan.

## Goals

1. Validate that CDP console and network event capture works as expected
2. Understand the full scope of CDP capabilities we need
3. Identify technical risks and unknowns
4. Define the project phases and their dependencies
5. Create a roadmap of concrete projects (P-003, P-004, etc.)

## Scope

In Scope:

- Proof-of-concept: CDP connection and event capture (console, network)
- Research: CDP protocol for all planned commands
- Design: Project breakdown and sequencing
- Documentation: Update DR-001 with any new findings

Out of Scope:

- Full implementation of any command
- IPC/daemon architecture implementation
- CLI framework setup
- Production-quality code

## Success Criteria

- [ ] Working proof-of-concept that captures console.log events via CDP
- [ ] Working proof-of-concept that captures network requests via CDP
- [ ] Documented understanding of CDP methods needed for each command
- [ ] Identified any commands that may be problematic or need redesign
- [ ] Created project documents for next 3-5 projects with clear sequencing
- [ ] Updated AGENTS.md with P-003 as active project

## Deliverables

- `poc/` directory with minimal Go code demonstrating CDP event capture
- `docs/research/cdp-commands.md` - mapping of webctl commands to CDP methods
- `docs/projects/p-003-*.md` through `docs/projects/p-00N-*.md` - next projects
- Updated DR-001 if any architectural changes needed

## Research Areas

CDP Event Capture:

- Runtime.consoleAPICalled - does it capture all console methods?
- Runtime.exceptionThrown - uncaught exceptions
- Network.requestWillBeSent, Network.responseReceived - full request/response cycle
- Network.loadingFinished - response bodies

CDP Commands:

- Page.navigate, Page.reload - navigation
- Runtime.evaluate - JS execution
- DOM.querySelector + Input.dispatchMouseEvent - clicking
- Input.insertText, Input.dispatchKeyEvent - typing
- Page.captureScreenshot - screenshots
- DOM.getOuterHTML - HTML extraction
- Network.getCookies, Network.setCookies - cookie management

Browser Launch:

- How to find Chrome/Chromium on different platforms
- Required launch flags for CDP (--remote-debugging-port, etc.)
- Headless vs headful considerations

## Questions & Uncertainties

- Can we get network response bodies reliably? (Some are streamed)
- How do we handle multiple frames/iframes?
- What happens if the page navigates while we're executing a command?
- Do we need to handle browser crashes/disconnects specially?
- Performance: is 10,000 network entries with bodies actually reasonable?

## Technical Approach

1. Create minimal Go program that:
   - Launches Chrome with CDP enabled
   - Connects via WebSocket
   - Subscribes to Runtime and Network events
   - Logs events to stdout

2. Test with a simple HTML page that:
   - Logs to console (log, warn, error)
   - Makes fetch requests
   - Throws an uncaught error

3. Document findings and any surprises

4. Based on findings, break remaining work into projects:
   - CDP core library
   - Daemon/IPC layer
   - CLI framework
   - Individual command groups
   - Testing/polish

## Notes

This project exists because we want to validate before committing to full implementation. If CDP event capture doesn't work as expected, we need to know early.

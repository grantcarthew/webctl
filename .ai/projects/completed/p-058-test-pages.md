# p-058: Test Pages

- Status: Done
- Started: 2026-01-16
- Completed: 2026-01-16
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create HTML test pages in testdata/pages/ for automated CLI testing. These pages provide controlled, predictable content for testing observation and interaction commands.

## Goals

1. Create test pages for forms, navigation, cookies, console, network, CSS, scrolling, and click targets
2. Ensure pages are simple and focused on specific test scenarios
3. Provide predictable content for assertions

## Scope

In Scope:

- testdata/pages/forms.html - form elements for type/select tests
- testdata/pages/navigation.html - links for back/forward tests
- testdata/pages/cookies.html - cookie setting/reading tests
- testdata/pages/console-types.html - log/warn/error/info console output
- testdata/pages/network-requests.html - fetch/XHR triggers
- testdata/pages/css-showcase.html - various CSS selectors and styles
- testdata/pages/slow-load.html - delayed content for ready command tests
- testdata/pages/scroll-long.html - tall page for scroll tests
- testdata/pages/click-targets.html - clickable elements with feedback

Out of Scope:

- Actual test scripts (p-059+)
- Complex JavaScript applications
- External dependencies

## Success Criteria

- [x] testdata/pages/ directory created
- [x] forms.html contains input, textarea, select, checkbox, radio elements
- [x] navigation.html contains links to other test pages
- [x] cookies.html has JavaScript to set/read cookies
- [x] console-types.html triggers console.log/warn/error/info on load
- [x] network-requests.html has buttons to trigger fetch requests
- [x] css-showcase.html demonstrates various CSS selectors
- [x] slow-load.html delays content rendering by configurable amount
- [x] scroll-long.html is taller than viewport with markers
- [x] click-targets.html has buttons that update page content on click
- [x] All pages are valid HTML5
- [x] All pages work with webctl serve

## Deliverables

- testdata/pages/forms.html
- testdata/pages/navigation.html
- testdata/pages/cookies.html
- testdata/pages/console-types.html
- testdata/pages/network-requests.html
- testdata/pages/css-showcase.html
- testdata/pages/slow-load.html
- testdata/pages/scroll-long.html
- testdata/pages/click-targets.html

## Technical Approach

Page structure:

- Simple, semantic HTML5
- Inline styles where needed (no external CSS)
- Minimal JavaScript (inline scripts only)
- Clear identifiable elements for selectors
- Predictable text content for assertions

Forms page elements:

- Text input with id="text-input"
- Email input with id="email-input"
- Password input with id="password-input"
- Textarea with id="textarea"
- Select dropdown with id="select"
- Checkbox with id="checkbox"
- Radio buttons with name="radio-group"
- Submit button with id="submit-btn"

Console types page:

- Automatically log different console levels on DOMContentLoaded
- Each log includes identifiable text

Network requests page:

- Button to trigger GET request
- Button to trigger POST request
- Status display element
- Use relative endpoints (`/api/echo`) — requires `webctl serve --proxy`

Slow-load page:

- Query parameter for delay: `?delay=2000` (default 1000ms)
- Append element after delay for `webctl ready` testing

## Current State

Repository context:

- `testdata/` directory exists with `index.html`, `backend.go`, `README.md`, `start-backend.sh`
- No `testdata/pages/` directory exists yet — needs to be created
- Test framework is complete: `test-runner` script and `scripts/bash_modules/` (assertions.sh, test-framework.sh, setup.sh)
- `context/rod/fixtures/` contains reference HTML patterns (minimal, functional test pages)
- `testdata/backend.go` provides HTTP endpoints for network testing: `/api/hello`, `/api/users`, `/api/echo`, `/status/{code}`, `/delay`

Reference patterns from Rod fixtures:

- `input.html` — form elements with event attributes for verification
- `click.html` — button with onclick that sets attribute for assertion
- `scroll.html`, `scroll-y.html` — elements with large margins to force scrolling
- `slow-render.html` — setTimeout to append element after delay
- `fetch.html` — fetch requests to relative endpoints

Key conventions:

- Use `onclick="this.setAttribute('event', 'value')"` pattern for click verification
- Use `onchange` events on form elements for input verification
- Relative paths for network requests (served via webctl serve)
- Simple inline styles (margins, dimensions) over decorative CSS

## Dependencies

- p-057: Test Runner (completed)

## Notes

- Keep pages simple and focused
- Use predictable IDs and classes for selectors
- Include visible feedback for interaction tests
- Ensure pages load quickly (minimal external resources)

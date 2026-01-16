# p-058: Test Pages

- Status: Pending
- Started:
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

- [ ] testdata/pages/ directory created
- [ ] forms.html contains input, textarea, select, checkbox, radio elements
- [ ] navigation.html contains links to other test pages
- [ ] cookies.html has JavaScript to set/read cookies
- [ ] console-types.html triggers console.log/warn/error/info on load
- [ ] network-requests.html has buttons to trigger fetch requests
- [ ] css-showcase.html demonstrates various CSS selectors
- [ ] slow-load.html delays content rendering by configurable amount
- [ ] scroll-long.html is taller than viewport with markers
- [ ] click-targets.html has buttons that update page content on click
- [ ] All pages are valid HTML5
- [ ] All pages work with webctl serve

## Deliverables

- testdata/pages/*.html (9 files)

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

## Dependencies

- p-057: Test Runner (completed)

## Notes

- Keep pages simple and focused
- Use predictable IDs and classes for selectors
- Include visible feedback for interaction tests
- Ensure pages load quickly (minimal external resources)

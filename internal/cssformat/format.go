package cssformat

import (
	"regexp"
	"strings"
)

// Format formats CSS with proper indentation for readability.
// Uses 2-space indentation and adds line breaks between rules.
func Format(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	var result strings.Builder
	var depth int
	const indent = "  " // 2 spaces

	// Track if we're inside a declaration block
	inBlock := false
	// Buffer for current line being built
	var line strings.Builder

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '{':
			// Opening brace - start of declaration block
			selector := strings.TrimSpace(line.String())
			if selector != "" {
				result.WriteString(strings.Repeat(indent, depth))
				result.WriteString(selector)
				result.WriteByte(' ')
			}
			result.WriteByte(ch)
			result.WriteByte('\n')
			line.Reset()
			depth++
			inBlock = true

		case '}':
			// Closing brace - end of declaration block
			if line.Len() > 0 {
				trimmed := strings.TrimSpace(line.String())
				if trimmed != "" && trimmed != ";" {
					result.WriteString(strings.Repeat(indent, depth))
					result.WriteString(trimmed)
					result.WriteByte('\n')
				}
				line.Reset()
			}
			depth--
			if depth < 0 {
				depth = 0 // Prevent negative depth from malformed CSS
			}
			result.WriteString(strings.Repeat(indent, depth))
			result.WriteByte(ch)
			result.WriteByte('\n')
			if depth == 0 {
				result.WriteByte('\n') // Blank line between top-level rules
			}
			inBlock = false

		case ';':
			// Semicolon - end of property declaration
			line.WriteByte(ch)
			if inBlock {
				trimmed := strings.TrimSpace(line.String())
				if trimmed != "" && trimmed != ";" {
					result.WriteString(strings.Repeat(indent, depth))
					result.WriteString(trimmed)
					result.WriteByte('\n')
				}
				line.Reset()
			}

		case '\n', '\r':
			// Skip existing newlines - we'll add our own
			continue

		case '\t':
			// Convert tabs to spaces
			line.WriteByte(' ')

		default:
			line.WriteByte(ch)
		}
	}

	// Flush any remaining content
	if line.Len() > 0 {
		trimmed := strings.TrimSpace(line.String())
		if trimmed != "" {
			result.WriteString(trimmed)
			result.WriteByte('\n')
		}
	}

	return strings.TrimSpace(result.String()) + "\n", nil
}

// FormatComputedStyles formats computed styles as CSS properties.
// Input: map of property names to values
// Output: formatted CSS properties (one per line)
func FormatComputedStyles(styles map[string]string) string {
	if len(styles) == 0 {
		return ""
	}

	var result strings.Builder

	// We don't sort - maintain the order from getComputedStyle
	for prop, value := range styles {
		result.WriteString(prop)
		result.WriteString(": ")
		result.WriteString(value)
		result.WriteString(";\n")
	}

	return result.String()
}

// FormatComputedStylesMulti formats multiple computed styles with -- separators.
// Input: slice of maps (one per element)
// Output: formatted CSS properties with -- separators between elements
func FormatComputedStylesMulti(stylesList []map[string]string) string {
	if len(stylesList) == 0 {
		return ""
	}

	var result strings.Builder

	for i, styles := range stylesList {
		if i > 0 {
			result.WriteString("--\n")
		}
		for prop, value := range styles {
			result.WriteString(prop)
			result.WriteString(": ")
			result.WriteString(value)
			result.WriteString(";\n")
		}
	}

	return result.String()
}

// CSSRule represents a parsed CSS rule with selector and body.
type CSSRule struct {
	Selector string
	Body     string
}

// ParseRules extracts CSS rules from CSS text.
// Returns a slice of CSSRule with selector and body.
func ParseRules(css string) []CSSRule {
	var rules []CSSRule

	// Simple state machine to extract rules
	// This handles nested braces (e.g., @media queries)
	var (
		currentSelector strings.Builder
		currentBody     strings.Builder
		depth           int
		inRule          bool
	)

	for i := 0; i < len(css); i++ {
		ch := css[i]

		switch ch {
		case '{':
			if depth == 0 {
				inRule = true
			}
			depth++
			if depth > 1 {
				currentBody.WriteByte(ch)
			}
		case '}':
			depth--
			if depth == 0 && inRule {
				// End of rule
				rules = append(rules, CSSRule{
					Selector: strings.TrimSpace(currentSelector.String()),
					Body:     strings.TrimSpace(currentBody.String()),
				})
				currentSelector.Reset()
				currentBody.Reset()
				inRule = false
			} else if depth > 0 {
				currentBody.WriteByte(ch)
			}
		default:
			if inRule {
				currentBody.WriteByte(ch)
			} else {
				currentSelector.WriteByte(ch)
			}
		}
	}

	return rules
}

// FilterRulesBySelector filters CSS rules to those whose selector matches the pattern.
// The pattern is matched case-insensitively against the selector text.
// Supports simple substring matching and basic patterns.
func FilterRulesBySelector(css, selectorPattern string) string {
	rules := ParseRules(css)
	if len(rules) == 0 {
		return ""
	}

	patternLower := strings.ToLower(selectorPattern)

	// Build regex for more flexible matching
	// This allows matching "h1" to find selectors like "h1", "div h1", ".class h1", etc.
	// Escape special regex chars in the pattern
	escaped := regexp.QuoteMeta(patternLower)
	// Match the pattern as a word boundary (allow for class/id prefixes)
	pattern := regexp.MustCompile(`(?i)(^|[\s,>+~])` + escaped + `($|[\s,>+~:\[.#])`)

	var result strings.Builder
	matchCount := 0

	for _, rule := range rules {
		selectorLower := strings.ToLower(rule.Selector)

		// Check if selector matches pattern
		// Either contains the pattern or matches the regex
		if strings.Contains(selectorLower, patternLower) || pattern.MatchString(rule.Selector) {
			if matchCount > 0 {
				result.WriteString("\n")
			}
			result.WriteString(rule.Selector)
			result.WriteString(" {\n")
			// Indent body
			bodyLines := strings.Split(rule.Body, ";")
			for _, line := range bodyLines {
				line = strings.TrimSpace(line)
				if line != "" {
					result.WriteString("  ")
					result.WriteString(line)
					result.WriteString(";\n")
				}
			}
			result.WriteString("}\n")
			matchCount++
		}
	}

	return result.String()
}

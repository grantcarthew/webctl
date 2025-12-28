package cssformat

import (
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

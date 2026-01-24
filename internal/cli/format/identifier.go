package format

import (
	"fmt"
	"strings"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// SanitizeIdentifier removes invalid CSS identifier characters.
// Only alphanumeric characters, hyphens, and underscores are retained.
// Returns an empty string if no valid characters remain.
//
// This function is used to clean id and class attributes before using them
// in CSS selector notation. Invalid characters are stripped to ensure the
// resulting identifier is valid CSS.
func SanitizeIdentifier(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// FormatElementIdentifier returns a CSS selector-style identifier for an element.
// The identifier helps users distinguish between multiple elements in output.
//
// Format:
//   - #id                if element has id attribute
//   - .class:N           if element has class attribute (N is 1-based index)
//   - tag:N              fallback using tag name (N is 1-based index)
//
// The index parameter is 0-based but displayed as 1-based (index+1) for user-friendliness.
// IDs and classes are sanitized to remove invalid CSS identifier characters.
// If sanitization results in an empty string, the next priority level is used.
//
// Examples:
//   - Element with id="header" -> "#header"
//   - Element with class="panel active" at index 0 -> ".panel:1"
//   - Element with tag "div" at index 2 -> "div:3"
func FormatElementIdentifier(meta ipc.ElementMeta, index int) string {
	if meta.ID != "" {
		sanitized := SanitizeIdentifier(meta.ID)
		if sanitized != "" {
			return "#" + sanitized
		}
	}
	if meta.Class != "" {
		sanitized := SanitizeIdentifier(meta.Class)
		if sanitized != "" {
			return fmt.Sprintf(".%s:%d", sanitized, index+1)
		}
	}
	return fmt.Sprintf("%s:%d", meta.Tag, index+1)
}

package htmlformat

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// Format formats HTML with proper indentation for readability.
// Uses 2-space indentation and preserves content in pre/textarea tags.
func Format(input string) (string, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	buf := &bytes.Buffer{}
	indentLevel := 0
	const indent = "  " // 2 spaces

	var rawTagStack []string // Track nested raw tags (pre, textarea)
	prevWasText := false
	needIndent := true

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		raw := string(tokenizer.Raw())
		inRawTag := len(rawTagStack) > 0

		switch tokenType {
		case html.DoctypeToken:
			if needIndent {
				buf.WriteString(strings.Repeat(indent, indentLevel))
			}
			buf.WriteString(raw)
			buf.WriteByte('\n')
			needIndent = true
			prevWasText = false

		case html.CommentToken:
			if needIndent && !inRawTag {
				buf.WriteString(strings.Repeat(indent, indentLevel))
			}
			buf.WriteString(raw)
			if !inRawTag {
				buf.WriteByte('\n')
				needIndent = true
			}
			prevWasText = false

		case html.StartTagToken:
			tagName := getTagName(tokenizer)
			isRawTag := isPreformatted(tagName)
			isVoid := isVoidElement(tagName)

			if needIndent && !inRawTag {
				buf.WriteString(strings.Repeat(indent, indentLevel))
			}
			buf.WriteString(raw)
			if !inRawTag {
				buf.WriteByte('\n')
			}
			needIndent = true
			prevWasText = false

			if isRawTag {
				rawTagStack = append(rawTagStack, tagName)
			}
			// Only increment indentation for non-void elements
			// Void elements have no closing tag, so incrementing would cause drift
			if !inRawTag && !isVoid {
				indentLevel++
			}

		case html.EndTagToken:
			tagName := getTagName(tokenizer)
			wasInRawTag := len(rawTagStack) > 0 && rawTagStack[len(rawTagStack)-1] == tagName

			// Decrement for normal tags OR when closing a raw tag
			// (raw tags increment when opened, so must decrement when closed)
			if !inRawTag || wasInRawTag {
				indentLevel--
			}

			if needIndent && !inRawTag {
				buf.WriteString(strings.Repeat(indent, indentLevel))
			}
			buf.WriteString(raw)
			// Add newline for normal tags OR when closing a raw tag
			// (so the next element starts on a new line)
			if !inRawTag || wasInRawTag {
				buf.WriteByte('\n')
			}
			needIndent = true
			prevWasText = false

			// Pop raw tag from stack
			if wasInRawTag {
				rawTagStack = rawTagStack[:len(rawTagStack)-1]
			}

		case html.SelfClosingTagToken:
			if needIndent && !inRawTag {
				buf.WriteString(strings.Repeat(indent, indentLevel))
			}
			buf.WriteString(raw)
			if !inRawTag {
				buf.WriteByte('\n')
			}
			needIndent = true
			prevWasText = false

		case html.TextToken:
			text := raw
			if inRawTag {
				// Preserve whitespace in raw tags
				buf.WriteString(text)
				needIndent = false
			} else {
				// Trim and collapse whitespace for normal text
				trimmed := strings.TrimSpace(text)
				// Also collapse multiple spaces within the text
				trimmed = collapseSpaces(trimmed)
				if trimmed != "" {
					if prevWasText {
						// Add space between consecutive text nodes
						buf.WriteByte(' ')
					} else if needIndent {
						buf.WriteString(strings.Repeat(indent, indentLevel))
					}
					buf.WriteString(trimmed)
					buf.WriteByte('\n')
					needIndent = true
					prevWasText = false
				}
			}
		}
	}

	return buf.String(), nil
}

// getTagName extracts the tag name from the tokenizer.
func getTagName(tokenizer *html.Tokenizer) string {
	name, _ := tokenizer.TagName()
	return string(name)
}

// isPreformatted checks if a tag should preserve whitespace.
// This includes pre, textarea, script, and style tags where formatting
// would break the content.
func isPreformatted(tagName string) bool {
	return tagName == "pre" ||
		tagName == "textarea" ||
		tagName == "script" ||
		tagName == "style"
}

// isVoidElement checks if a tag is a void element (no closing tag in HTML5).
// These elements cannot have children and don't need/have closing tags.
func isVoidElement(tagName string) bool {
	switch tagName {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

// collapseSpaces collapses multiple consecutive spaces into a single space.
func collapseSpaces(s string) string {
	var result strings.Builder
	prevWasSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevWasSpace {
				result.WriteByte(' ')
				prevWasSpace = true
			}
		} else {
			result.WriteRune(r)
			prevWasSpace = false
		}
	}
	return result.String()
}

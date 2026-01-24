package format

import (
	"fmt"
	"io"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// ComputedStyles outputs computed styles in text format.
// Format: property: value (one per line)
func ComputedStyles(w io.Writer, styles map[string]string) error {
	for prop, value := range styles {
		_, err := fmt.Fprintf(w, "%s: %s\n", prop, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// ComputedStylesMulti outputs computed styles for multiple elements.
// Each element is identified by a CSS selector-style identifier (#id, .class:N, or tag:N)
// printed on a separate line before its styles. Multiple elements are separated by "--".
//
// Format:
//
//	#header
//	color: blue
//	font-size: 16px
//	--
//	.panel:2
//	background: white
//
// Elements are identified by priority: ID > first class > tag name.
// Index numbers are 1-based for user readability.
func ComputedStylesMulti(w io.Writer, elements []ipc.ElementWithStyles) error {
	for i, elem := range elements {
		if i > 0 {
			if _, err := fmt.Fprintln(w, ipc.MultiElementSeparator); err != nil {
				return err
			}
		}
		// Output element identifier
		identifier := FormatElementIdentifier(elem.ElementMeta, i)
		if _, err := fmt.Fprintln(w, identifier); err != nil {
			return err
		}
		// Output styles
		for prop, value := range elem.Styles {
			if _, err := fmt.Fprintf(w, "%s: %s\n", prop, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// PropertyValue outputs a single CSS property value.
func PropertyValue(w io.Writer, value string) error {
	_, err := fmt.Fprintln(w, value)
	return err
}

// InlineStyles outputs inline style attributes for multiple elements.
// Each element is identified by a CSS selector-style identifier (#id, .class:N, or tag:N)
// printed on a separate line before its inline style. Multiple elements are separated by "--".
// Empty inline styles are displayed as "(empty)".
//
// Format:
//
//	#main
//	color: red; margin: 10px;
//	--
//	.item:1
//	(empty)
//
// Elements are identified by priority: ID > first class > tag name.
// Index numbers are 1-based for user readability.
func InlineStyles(w io.Writer, elements []ipc.ElementWithStyles) error {
	for i, elem := range elements {
		if i > 0 {
			if _, err := fmt.Fprintln(w, ipc.MultiElementSeparator); err != nil {
				return err
			}
		}
		// Output element identifier
		identifier := FormatElementIdentifier(elem.ElementMeta, i)
		if _, err := fmt.Fprintln(w, identifier); err != nil {
			return err
		}
		// Output inline style
		if elem.Inline == "" {
			if _, err := fmt.Fprintln(w, "(empty)"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, elem.Inline); err != nil {
				return err
			}
		}
	}
	return nil
}

// MatchedRules outputs matched CSS rules with -- separators.
func MatchedRules(w io.Writer, rules []ipc.CSSMatchedRule) error {
	for i, rule := range rules {
		if i > 0 {
			if _, err := fmt.Fprintln(w, ipc.MultiElementSeparator); err != nil {
				return err
			}
		}
		// Output selector as comment, with source if inherited
		if rule.Source == "inherited" {
			if _, err := fmt.Fprintf(w, "/* %s (inherited) */\n", rule.Selector); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "/* %s */\n", rule.Selector); err != nil {
				return err
			}
		}
		// Output properties
		for prop, value := range rule.Properties {
			if _, err := fmt.Fprintf(w, "%s: %s;\n", prop, value); err != nil {
				return err
			}
		}
	}
	return nil
}

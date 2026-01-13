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

// ComputedStylesMulti outputs computed styles for multiple elements with -- separators.
func ComputedStylesMulti(w io.Writer, stylesList []map[string]string) error {
	for i, styles := range stylesList {
		if i > 0 {
			if _, err := fmt.Fprintln(w, "--"); err != nil {
				return err
			}
		}
		for prop, value := range styles {
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

// InlineStyles outputs inline style attributes with -- separators.
func InlineStyles(w io.Writer, styles []string) error {
	for i, style := range styles {
		if i > 0 {
			if _, err := fmt.Fprintln(w, "--"); err != nil {
				return err
			}
		}
		if style == "" {
			if _, err := fmt.Fprintln(w, "(empty)"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, style); err != nil {
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
			if _, err := fmt.Fprintln(w, "--"); err != nil {
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

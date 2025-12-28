package format

import (
	"fmt"
	"io"
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

// PropertyValue outputs a single CSS property value.
func PropertyValue(w io.Writer, value string) error {
	_, err := fmt.Fprintln(w, value)
	return err
}

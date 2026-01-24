package cli

import (
	"testing"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestFormatHTMLElementIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		meta     ipc.ElementMeta
		index    int
		expected string
	}{
		{
			name:     "element with id",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "element with id (ignores class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header", Class: "panel"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "element with class only",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "element with class at index 2",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel"},
			index:    2,
			expected: ".panel:3",
		},
		{
			name:     "element with tag only",
			meta:     ipc.ElementMeta{Tag: "div"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "element with tag at index 5",
			meta:     ipc.ElementMeta{Tag: "span"},
			index:    5,
			expected: "span:6",
		},
		{
			name:     "id with special chars",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header@#$"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "id with only special chars (falls back to class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "@#$", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "id with only special chars (falls back to tag)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "@#$"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "class with special chars",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel@#$"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "class with only special chars (falls back to tag)",
			meta:     ipc.ElementMeta{Tag: "div", Class: "@#$"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "empty id (uses class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "empty class (uses tag)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "", Class: ""},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "id with spaces",
			meta:     ipc.ElementMeta{Tag: "div", ID: "my header"},
			index:    0,
			expected: "#myheader",
		},
		{
			name:     "class with hyphens",
			meta:     ipc.ElementMeta{Tag: "div", Class: "my-panel"},
			index:    0,
			expected: ".my-panel:1",
		},
		{
			name:     "class with underscores",
			meta:     ipc.ElementMeta{Tag: "div", Class: "my_panel"},
			index:    0,
			expected: ".my_panel:1",
		},
		{
			name:     "svg element",
			meta:     ipc.ElementMeta{Tag: "svg", ID: "icon"},
			index:    0,
			expected: "#icon",
		},
		{
			name:     "paragraph with class",
			meta:     ipc.ElementMeta{Tag: "p", Class: "highlight"},
			index:    3,
			expected: ".highlight:4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format.FormatElementIdentifier(tt.meta, tt.index)
			if got != tt.expected {
				t.Errorf("format.FormatElementIdentifier(%+v, %d) = %q, want %q", tt.meta, tt.index, got, tt.expected)
			}
		})
	}
}

func TestSanitizeSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "id selector",
			input:    "#header",
			expected: "header",
		},
		{
			name:     "class selector",
			input:    ".panel",
			expected: "panel",
		},
		{
			name:     "complex selector",
			input:    "div.panel#header",
			expected: "div-panel-header",
		},
		{
			name:     "with spaces",
			input:    "div > span.class",
			expected: "div-span-class",
		},
		{
			name:     "long selector",
			input:    "this-is-a-very-long-selector-name-that-exceeds-thirty-characters",
			expected: "this-is-a-very-long-selector-n",
		},
		{
			name:     "empty after sanitization",
			input:    "###",
			expected: "element",
		},
		{
			name:     "brackets and attributes",
			input:    "input[type='text']",
			expected: "input-type-text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSelector(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeSelector(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

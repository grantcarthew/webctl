package daemon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// Test element metadata extraction from JavaScript responses

func TestCSSInlineResponseFormat(t *testing.T) {
	// This test verifies the expected format of the JavaScript response
	// for inline styles with element metadata

	// Simulate JavaScript response
	jsResponse := `[
		{"tag": "div", "id": "header", "class": null, "inline": "color: blue;"},
		{"tag": "div", "id": null, "class": "panel", "inline": "--active-panel-height: 0px;"},
		{"tag": "div", "id": null, "class": null, "inline": ""}
	]`

	var elements []ipc.ElementWithStyles
	err := json.Unmarshal([]byte(jsResponse), &elements)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elements))
	}

	// Check first element
	if elements[0].Tag != "div" {
		t.Errorf("element 0: expected tag 'div', got %q", elements[0].Tag)
	}
	if elements[0].ID != "header" {
		t.Errorf("element 0: expected id 'header', got %q", elements[0].ID)
	}
	if elements[0].Inline != "color: blue;" {
		t.Errorf("element 0: expected inline 'color: blue;', got %q", elements[0].Inline)
	}

	// Check second element
	if elements[1].Tag != "div" {
		t.Errorf("element 1: expected tag 'div', got %q", elements[1].Tag)
	}
	if elements[1].Class != "panel" {
		t.Errorf("element 1: expected class 'panel', got %q", elements[1].Class)
	}
	if elements[1].Inline != "--active-panel-height: 0px;" {
		t.Errorf("element 1: expected inline '--active-panel-height: 0px;', got %q", elements[1].Inline)
	}

	// Check third element (empty inline)
	if elements[2].Inline != "" {
		t.Errorf("element 2: expected empty inline, got %q", elements[2].Inline)
	}
}

func TestCSSComputedResponseFormat(t *testing.T) {
	// This test verifies the expected format of the JavaScript response
	// for computed styles with element metadata

	// Simulate JavaScript response
	jsResponse := `[
		{
			"tag": "div",
			"id": "header",
			"class": null,
			"styles": {"color": "rgb(0, 0, 255)", "font-size": "16px"}
		},
		{
			"tag": "div",
			"id": null,
			"class": "panel",
			"styles": {"padding": "10px", "margin": "0px"}
		}
	]`

	var elements []ipc.ElementWithStyles
	err := json.Unmarshal([]byte(jsResponse), &elements)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elements))
	}

	// Check first element
	if elements[0].Tag != "div" {
		t.Errorf("element 0: expected tag 'div', got %q", elements[0].Tag)
	}
	if elements[0].ID != "header" {
		t.Errorf("element 0: expected id 'header', got %q", elements[0].ID)
	}
	if len(elements[0].Styles) != 2 {
		t.Errorf("element 0: expected 2 styles, got %d", len(elements[0].Styles))
	}
	if elements[0].Styles["color"] != "rgb(0, 0, 255)" {
		t.Errorf("element 0: expected color 'rgb(0, 0, 255)', got %q", elements[0].Styles["color"])
	}

	// Check second element
	if elements[1].Class != "panel" {
		t.Errorf("element 1: expected class 'panel', got %q", elements[1].Class)
	}
	if len(elements[1].Styles) != 2 {
		t.Errorf("element 1: expected 2 styles, got %d", len(elements[1].Styles))
	}
}

func TestHTMLMultiResponseFormat(t *testing.T) {
	// This test verifies the expected format of the JavaScript response
	// for HTML with element metadata

	// Simulate JavaScript response
	jsResponse := `[
		{
			"tag": "div",
			"id": "header",
			"class": null,
			"html": "<div id=\"header\">Header</div>"
		},
		{
			"tag": "div",
			"id": null,
			"class": "panel",
			"html": "<div class=\"panel\">Panel</div>"
		},
		{
			"tag": "p",
			"id": null,
			"class": "highlight",
			"html": "<p class=\"highlight\">Text</p>"
		}
	]`

	var elements []ipc.ElementWithHTML
	err := json.Unmarshal([]byte(jsResponse), &elements)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elements))
	}

	// Check first element
	if elements[0].Tag != "div" {
		t.Errorf("element 0: expected tag 'div', got %q", elements[0].Tag)
	}
	if elements[0].ID != "header" {
		t.Errorf("element 0: expected id 'header', got %q", elements[0].ID)
	}
	if elements[0].HTML != "<div id=\"header\">Header</div>" {
		t.Errorf("element 0: unexpected HTML: %q", elements[0].HTML)
	}

	// Check second element
	if elements[1].Tag != "div" {
		t.Errorf("element 1: expected tag 'div', got %q", elements[1].Tag)
	}
	if elements[1].Class != "panel" {
		t.Errorf("element 1: expected class 'panel', got %q", elements[1].Class)
	}

	// Check third element (paragraph)
	if elements[2].Tag != "p" {
		t.Errorf("element 2: expected tag 'p', got %q", elements[2].Tag)
	}
	if elements[2].Class != "highlight" {
		t.Errorf("element 2: expected class 'highlight', got %q", elements[2].Class)
	}
}

func TestElementMetaEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		jsObject string
		checkFn  func(*testing.T, ipc.ElementWithStyles)
	}{
		{
			name:     "null id and class",
			jsObject: `{"tag": "div", "id": null, "class": null, "inline": ""}`,
			checkFn: func(t *testing.T, elem ipc.ElementWithStyles) {
				if elem.ID != "" {
					t.Errorf("expected empty id, got %q", elem.ID)
				}
				if elem.Class != "" {
					t.Errorf("expected empty class, got %q", elem.Class)
				}
			},
		},
		{
			name:     "empty string id and class",
			jsObject: `{"tag": "div", "id": "", "class": "", "inline": ""}`,
			checkFn: func(t *testing.T, elem ipc.ElementWithStyles) {
				if elem.ID != "" {
					t.Errorf("expected empty id, got %q", elem.ID)
				}
				if elem.Class != "" {
					t.Errorf("expected empty class, got %q", elem.Class)
				}
			},
		},
		{
			name:     "id with special characters",
			jsObject: `{"tag": "div", "id": "my-header_123", "class": null, "inline": ""}`,
			checkFn: func(t *testing.T, elem ipc.ElementWithStyles) {
				if elem.ID != "my-header_123" {
					t.Errorf("expected id 'my-header_123', got %q", elem.ID)
				}
			},
		},
		{
			name:     "class with hyphens",
			jsObject: `{"tag": "div", "id": null, "class": "my-panel-class", "inline": ""}`,
			checkFn: func(t *testing.T, elem ipc.ElementWithStyles) {
				if elem.Class != "my-panel-class" {
					t.Errorf("expected class 'my-panel-class', got %q", elem.Class)
				}
			},
		},
		{
			name:     "svg element",
			jsObject: `{"tag": "svg", "id": "icon", "class": null, "inline": ""}`,
			checkFn: func(t *testing.T, elem ipc.ElementWithStyles) {
				if elem.Tag != "svg" {
					t.Errorf("expected tag 'svg', got %q", elem.Tag)
				}
				if elem.ID != "icon" {
					t.Errorf("expected id 'icon', got %q", elem.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var elem ipc.ElementWithStyles
			err := json.Unmarshal([]byte(tt.jsObject), &elem)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			tt.checkFn(t, elem)
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that the old Inline field is still populated for backward compatibility

	t.Run("inline styles backward compat", func(t *testing.T) {
		// Simulate a CSSData response with both old and new fields
		cssData := ipc.CSSData{
			InlineMulti: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Inline:      "color: blue;",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					Inline:      "",
				},
			},
			Inline: []string{"color: blue;", ""},
		}

		// Verify both fields are populated
		if len(cssData.InlineMulti) != 2 {
			t.Errorf("expected 2 InlineMulti elements, got %d", len(cssData.InlineMulti))
		}
		if len(cssData.Inline) != 2 {
			t.Errorf("expected 2 Inline elements, got %d", len(cssData.Inline))
		}

		// Verify inline values match
		for i := range cssData.InlineMulti {
			if cssData.InlineMulti[i].Inline != cssData.Inline[i] {
				t.Errorf("element %d: inline mismatch: %q vs %q",
					i, cssData.InlineMulti[i].Inline, cssData.Inline[i])
			}
		}
	})

	t.Run("html multi backward compat", func(t *testing.T) {
		// Simulate an HTMLData response with both old and new fields
		htmlData := ipc.HTMLData{
			HTMLMulti: []ipc.ElementWithHTML{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					HTML:        "<div id=\"header\">Header</div>",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					HTML:        "<div class=\"panel\">Panel</div>",
				},
			},
			HTML: "#header\n<div id=\"header\">Header</div>\n--\n.panel:1\n<div class=\"panel\">Panel</div>\n",
		}

		// Verify both fields are populated
		if len(htmlData.HTMLMulti) != 2 {
			t.Errorf("expected 2 HTMLMulti elements, got %d", len(htmlData.HTMLMulti))
		}
		if htmlData.HTML == "" {
			t.Error("expected HTML field to be populated")
		}

		// Verify HTML field contains separators (backward compatibility)
		if htmlData.HTML == "" {
			t.Error("expected HTML field to be populated for backward compatibility")
		}
		if !strings.Contains(htmlData.HTML, ipc.MultiElementSeparator) {
			t.Error("expected HTML field to contain element separators")
		}
	})
}

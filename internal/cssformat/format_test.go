package cssformat

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty CSS",
			input: "",
			want:  "",
		},
		{
			name:  "simple rule",
			input: "body{margin:0;padding:0;}",
			want:  "body {\n  margin:0;\n  padding:0;\n}\n",
		},
		{
			name:  "multiple rules",
			input: "body{margin:0;}.header{display:flex;}",
			want:  "body {\n  margin:0;\n}\n\n.header {\n  display:flex;\n}\n",
		},
		{
			name:  "already formatted",
			input: "body {\n  margin: 0;\n}\n",
			want:  "body {\n  margin: 0;\n}\n",
		},
		{
			name:  "nested rule (media query)",
			input: "@media (max-width:768px){.container{width:100%;}}",
			want:  "@media (max-width:768px) {\n  .container {\n    width:100%;\n  }\n}\n",
		},
		{
			name:  "CSS comments",
			input: "/* Global styles */body{margin:0;}/* Header */h1{font-size:2em;}",
			want:  "/* Global styles */body {\n  margin:0;\n}\n\n/* Header */h1 {\n  font-size:2em;\n}\n",
		},
		{
			name:  "multiple selectors",
			input: "h1,h2,h3{font-family:sans-serif;color:#333;}",
			want:  "h1,h2,h3 {\n  font-family:sans-serif;\n  color:#333;\n}\n",
		},
		{
			name:  "pseudo-classes",
			input: "a:hover{color:red;}button::before{content:'→';}",
			want:  "a:hover {\n  color:red;\n}\n\nbutton::before {\n  content:'→';\n}\n",
		},
		{
			name:  "attribute selectors",
			input: "[data-active='true']{background:blue;}input[type='text']{border:1px solid;}",
			want:  "[data-active='true'] {\n  background:blue;\n}\n\ninput[type='text'] {\n  border:1px solid;\n}\n",
		},
		{
			name:  "!important declarations",
			input: ".override{color:red!important;margin:0!important;}",
			want:  ".override {\n  color:red!important;\n  margin:0!important;\n}\n",
		},
		{
			name:  "CSS functions",
			input: ".box{width:calc(100% - 20px);background:var(--primary);background-image:url('image.png');}",
			want:  ".box {\n  width:calc(100% - 20px);\n  background:var(--primary);\n  background-image:url('image.png');\n}\n",
		},
		{
			name:  "vendor prefixes",
			input: ".transform{-webkit-transform:rotate(45deg);-moz-transform:rotate(45deg);transform:rotate(45deg);}",
			want:  ".transform {\n  -webkit-transform:rotate(45deg);\n  -moz-transform:rotate(45deg);\n  transform:rotate(45deg);\n}\n",
		},
		{
			name:  "empty rule",
			input: "body{}",
			want:  "body {\n}\n",
		},
		{
			name:  "@keyframes animation",
			input: "@keyframes slide{0%{left:0;}100%{left:100%;}}",
			want:  "@keyframes slide {\n  0% {\n    left:0;\n  }\n  100% {\n    left:100%;\n  }\n}\n",
		},
		{
			name:  "@font-face declaration",
			input: "@font-face{font-family:'Custom';src:url('font.woff2');}",
			want:  "@font-face {\n  font-family:'Custom';\n  src:url('font.woff2');\n}\n",
		},
		{
			name:  "CSS custom properties",
			input: ":root{--primary:#007bff;--spacing:1rem;}.btn{color:var(--primary);padding:var(--spacing);}",
			want:  ":root {\n  --primary:#007bff;\n  --spacing:1rem;\n}\n\n.btn {\n  color:var(--primary);\n  padding:var(--spacing);\n}\n",
		},
		{
			name:  "minified with no spaces",
			input: "body{margin:0;padding:0;background:#fff;color:#000;font:16px/1.5 sans-serif;}",
			want:  "body {\n  margin:0;\n  padding:0;\n  background:#fff;\n  color:#000;\n  font:16px/1.5 sans-serif;\n}\n",
		},
		{
			name:  "deeply nested media queries",
			input: "@media screen{@media (min-width:768px){.container{max-width:750px;}}}",
			want:  "@media screen {\n  @media (min-width:768px) {\n    .container {\n      max-width:750px;\n    }\n  }\n}\n",
		},
		{
			name:  "child and descendant combinators",
			input: "nav>ul>li{display:inline;}article p{margin:1em;}",
			want:  "nav>ul>li {\n  display:inline;\n}\n\narticle p {\n  margin:1em;\n}\n",
		},
		{
			name:  "adjacent sibling combinator",
			input: "h1+p{font-weight:bold;}",
			want:  "h1+p {\n  font-weight:bold;\n}\n",
		},
		{
			name:  "@supports rule",
			input: "@supports (display:grid){.grid{display:grid;}}",
			want:  "@supports (display:grid) {\n  .grid {\n    display:grid;\n  }\n}\n",
		},
		{
			name:  "complex selector list",
			input: "header nav ul li a:not(.active):hover,.btn:focus{outline:2px solid blue;}",
			want:  "header nav ul li a:not(.active):hover,.btn:focus {\n  outline:2px solid blue;\n}\n",
		},
		{
			name:  "unicode content",
			input: ".icon::before{content:'\\2192';}",
			want:  ".icon::before {\n  content:'\\2192';\n}\n",
		},
		{
			name:  "grid template areas",
			input: ".layout{display:grid;grid-template-areas:'header header' 'sidebar main' 'footer footer';}",
			want:  ".layout {\n  display:grid;\n  grid-template-areas:'header header' 'sidebar main' 'footer footer';\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format(tt.input)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Format() =\n%q\nwant:\n%q", got, tt.want)
				// Show line-by-line comparison for debugging
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(tt.want, "\n")
				for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
					gl := ""
					wl := ""
					if i < len(gotLines) {
						gl = gotLines[i]
					}
					if i < len(wantLines) {
						wl = wantLines[i]
					}
					if gl != wl {
						t.Logf("Line %d differs:\n  got:  %q\n  want: %q", i, gl, wl)
					}
				}
			}
		})
	}
}

func TestFormatComputedStyles(t *testing.T) {
	tests := []struct {
		name         string
		styles       map[string]string
		want         string
		wantContains string
	}{
		{
			name:   "empty styles",
			styles: map[string]string{},
			want:   "",
		},
		{
			name: "single property",
			styles: map[string]string{
				"display": "flex",
			},
			wantContains: "display: flex;",
		},
		{
			name: "multiple properties",
			styles: map[string]string{
				"display":          "flex",
				"background-color": "rgb(255, 255, 255)",
				"width":            "1200px",
			},
			wantContains: "display: flex;",
		},
		{
			name: "complex values",
			styles: map[string]string{
				"transform":     "matrix(1, 0, 0, 1, 0, 0)",
				"box-shadow":    "0px 2px 4px rgba(0, 0, 0, 0.1)",
				"grid-template": "auto / 1fr 1fr 1fr",
			},
			wantContains: "transform: matrix(1, 0, 0, 1, 0, 0);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatComputedStyles(tt.styles)
			if tt.want != "" && got != tt.want {
				t.Errorf("FormatComputedStyles() = %q, want %q", got, tt.want)
			}
			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("FormatComputedStyles() = %q, should contain %q", got, tt.wantContains)
			}
		})
	}
}

// Edge case tests
func TestFormat_EdgeCases(t *testing.T) {
	t.Run("malformed CSS - missing closing brace", func(t *testing.T) {
		input := "body{margin:0;padding:0;"
		// Should not panic
		_, err := Format(input)
		if err != nil {
			t.Fatalf("Format() should handle malformed CSS, got error: %v", err)
		}
	})

	t.Run("malformed CSS - extra closing brace", func(t *testing.T) {
		input := "body{margin:0;}}"
		// Should not panic
		_, err := Format(input)
		if err != nil {
			t.Fatalf("Format() should handle malformed CSS, got error: %v", err)
		}
	})

	t.Run("CSS with tabs", func(t *testing.T) {
		input := "body{\tmargin:0;\tpadding:0;\t}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Tabs should be converted to spaces
		if strings.Contains(result, "\t") {
			t.Error("Tabs should be converted to spaces")
		}
	})

	t.Run("CSS with Windows line endings", func(t *testing.T) {
		input := "body {\r\n  margin: 0;\r\n}\r\n"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Should handle CRLF gracefully
		if !strings.Contains(result, "margin") {
			t.Error("Content lost with CRLF line endings")
		}
	})

	t.Run("very long property value", func(t *testing.T) {
		longValue := "url('data:image/png;base64," + strings.Repeat("A", 10000) + "')"
		input := ".bg{background-image:" + longValue + ";}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Should not crash with long values - exact preservation is not guaranteed
		// Just check that we got some output and the key parts are there
		if !strings.Contains(result, ".bg") || !strings.Contains(result, "background-image") {
			t.Error("Long property value caused formatting to fail")
		}
		if !strings.Contains(result, "AAAA") {
			t.Error("Long property value was completely lost")
		}
	})

	t.Run("comment inside rule", func(t *testing.T) {
		input := "body{margin:0;/* spacing */padding:0;}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Comment should be preserved
		if !strings.Contains(result, "/* spacing */") {
			t.Error("Comment inside rule not preserved")
		}
	})

	t.Run("escaped characters in selector", func(t *testing.T) {
		input := ".my\\:component{color:red;}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Escaped characters should be preserved
		if !strings.Contains(result, "my\\:component") {
			t.Error("Escaped character in selector not preserved")
		}
	})

	t.Run("multiple blank lines", func(t *testing.T) {
		input := "body{margin:0;}\n\n\n\n.header{padding:0;}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Should consolidate to single blank line between rules
		if strings.Contains(result, "\n\n\n") {
			t.Error("Multiple blank lines not consolidated")
		}
	})

	t.Run("CSS with Unicode", func(t *testing.T) {
		input := ".emoji::before{content:'😀 🎉 ✨';}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		// Unicode should be preserved
		if !strings.Contains(result, "😀") || !strings.Contains(result, "🎉") {
			t.Error("Unicode characters not preserved")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result, err := Format("")
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		if result != "" {
			t.Errorf("Format(\"\") = %q, want \"\"", result)
		}
	})

	t.Run("only whitespace", func(t *testing.T) {
		input := "   \n\t  \r\n  "
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		trimmed := strings.TrimSpace(result)
		if trimmed != "" {
			t.Errorf("Format(whitespace) should produce empty output, got %q", result)
		}
	})

	t.Run("single selector no properties", func(t *testing.T) {
		input := "body{}"
		result, err := Format(input)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}
		expected := "body {\n}\n"
		if result != expected {
			t.Errorf("Format() = %q, want %q", result, expected)
		}
	})
}

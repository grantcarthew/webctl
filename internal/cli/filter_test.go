package cli

import (
	"strings"
	"testing"
)

func TestFilterHTMLByText(t *testing.T) {
	// Sample HTML content (10 lines)
	html := `<html>
<head>
<title>Test Page</title>
</head>
<body>
<div class="header">Header</div>
<div class="content">Content with login form</div>
<div class="footer">Footer</div>
</body>
</html>`

	tests := []struct {
		name       string
		searchText string
		before     int
		after      int
		wantLines  []string
		wantErr    bool
	}{
		{
			name:       "no context - single match",
			searchText: "login",
			before:     0,
			after:      0,
			wantLines:  []string{`<div class="content">Content with login form</div>`},
		},
		{
			name:       "no context - multiple matches",
			searchText: "div",
			before:     0,
			after:      0,
			wantLines: []string{
				`<div class="header">Header</div>`,
				`<div class="content">Content with login form</div>`,
				`<div class="footer">Footer</div>`,
			},
		},
		{
			name:       "with before context",
			searchText: "login",
			before:     2,
			after:      0,
			wantLines: []string{
				`<body>`,
				`<div class="header">Header</div>`,
				`<div class="content">Content with login form</div>`,
			},
		},
		{
			name:       "with after context",
			searchText: "login",
			before:     0,
			after:      2,
			wantLines: []string{
				`<div class="content">Content with login form</div>`,
				`<div class="footer">Footer</div>`,
				`</body>`,
			},
		},
		{
			name:       "with both before and after context",
			searchText: "login",
			before:     1,
			after:      1,
			wantLines: []string{
				`<div class="header">Header</div>`,
				`<div class="content">Content with login form</div>`,
				`<div class="footer">Footer</div>`,
			},
		},
		{
			name:       "context at start of file",
			searchText: "<html>",
			before:     5,
			after:      1,
			wantLines: []string{
				`<html>`,
				`<head>`,
			},
		},
		{
			name:       "context at end of file",
			searchText: "</html>",
			before:     1,
			after:      5,
			wantLines: []string{
				`</body>`,
				`</html>`,
			},
		},
		{
			name:       "overlapping contexts merge",
			searchText: "div",
			before:     1,
			after:      1,
			wantLines: []string{
				`<body>`,
				`<div class="header">Header</div>`,
				`<div class="content">Content with login form</div>`,
				`<div class="footer">Footer</div>`,
				`</body>`,
			},
		},
		{
			name:       "case insensitive",
			searchText: "LOGIN",
			before:     0,
			after:      0,
			wantLines:  []string{`<div class="content">Content with login form</div>`},
		},
		{
			name:       "no matches",
			searchText: "nonexistent",
			before:     0,
			after:      0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterHTMLByText(html, tt.searchText, tt.before, tt.after)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := strings.Join(tt.wantLines, "\n")
			if result != want {
				t.Errorf("result mismatch:\ngot:\n%s\n\nwant:\n%s", result, want)
			}
		})
	}
}

func TestFilterHTMLByTextNonContiguousRanges(t *testing.T) {
	// Content with matches far apart
	html := `line1
line2
line3 match
line4
line5
line6
line7
line8
line9 match
line10`

	result, err := filterHTMLByText(html, "match", 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have separator between non-contiguous ranges
	if !strings.Contains(result, "--") {
		t.Error("expected separator between non-contiguous ranges")
	}

	lines := strings.Split(result, "\n")
	// Should have: line2, line3 match, line4, --, line8, line9 match, line10
	if len(lines) != 7 {
		t.Errorf("expected 7 lines (including separator), got %d: %v", len(lines), lines)
	}

	if lines[3] != "--" {
		t.Errorf("expected separator at line 4, got: %s", lines[3])
	}
}

func TestFilterCSSByText(t *testing.T) {
	// Sample CSS content
	css := `.header {
  background: blue;
  color: white;
}
.content {
  background: white;
  color: black;
}
.footer {
  background: gray;
  color: white;
}`

	tests := []struct {
		name       string
		searchText string
		before     int
		after      int
		wantLines  []string
		wantErr    bool
	}{
		{
			name:       "no context - single property match",
			searchText: "gray",
			before:     0,
			after:      0,
			wantLines:  []string{`  background: gray;`},
		},
		{
			name:       "no context - multiple matches",
			searchText: "white",
			before:     0,
			after:      0,
			wantLines: []string{
				`  color: white;`,
				`--`,
				`  background: white;`,
				`--`,
				`  color: white;`,
			},
		},
		{
			name:       "with context to capture full rule",
			searchText: "gray",
			before:     1,
			after:      2,
			wantLines: []string{
				`.footer {`,
				`  background: gray;`,
				`  color: white;`,
				`}`,
			},
		},
		{
			name:       "case insensitive",
			searchText: "BACKGROUND",
			before:     0,
			after:      0,
			wantLines: []string{
				`  background: blue;`,
				`--`,
				`  background: white;`,
				`--`,
				`  background: gray;`,
			},
		},
		{
			name:       "no matches",
			searchText: "nonexistent",
			before:     0,
			after:      0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterCSSByText(css, tt.searchText, tt.before, tt.after)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := strings.Join(tt.wantLines, "\n")
			if result != want {
				t.Errorf("result mismatch:\ngot:\n%s\n\nwant:\n%s", result, want)
			}
		})
	}
}

func TestFilterCSSByTextNonContiguousRanges(t *testing.T) {
	// CSS with rules far apart
	css := `.a { color: red; }
.b { color: blue; }
.c { color: green; }
.d { color: yellow; }
.e { color: red; }
.f { color: purple; }`

	result, err := filterCSSByText(css, "red", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have separator between non-contiguous ranges
	if !strings.Contains(result, "--") {
		t.Error("expected separator between non-contiguous ranges")
	}
}

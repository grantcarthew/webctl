package markdownformat

import (
	"strings"
	"testing"
)

func TestConvertTable(t *testing.T) {
	html := `<table>
		<thead><tr><th>Name</th><th>Age</th></tr></thead>
		<tbody>
			<tr><td>Alice</td><td>30</td></tr>
			<tr><td>Bob</td><td>25</td></tr>
		</tbody>
	</table>`

	md, err := Convert(html)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	lines := splitNonEmpty(md)
	if len(lines) < 4 {
		t.Fatalf("expected a header, separator, and two data rows, got:\n%s", md)
	}

	// Header row carries both column names as pipe-delimited cells.
	if !strings.Contains(lines[0], "Name") || !strings.Contains(lines[0], "Age") {
		t.Errorf("header row missing columns: %q", lines[0])
	}
	if strings.Count(lines[0], "|") < 3 {
		t.Errorf("header row not pipe-delimited: %q", lines[0])
	}

	// Second line is the GitHub-flavored separator (dashes between pipes).
	if !strings.Contains(lines[1], "-") || strings.Count(lines[1], "|") < 3 {
		t.Errorf("missing table separator row: %q", lines[1])
	}

	if !strings.Contains(md, "Alice") || !strings.Contains(md, "30") {
		t.Errorf("table missing Alice/30 row:\n%s", md)
	}
	if !strings.Contains(md, "Bob") || !strings.Contains(md, "25") {
		t.Errorf("table missing Bob/25 row:\n%s", md)
	}
}

func TestConvertNestedList(t *testing.T) {
	html := `<ul>
		<li>One
			<ul>
				<li>Nested A</li>
				<li>Nested B</li>
			</ul>
		</li>
		<li>Two</li>
	</ul>`

	md, err := Convert(html)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	for _, want := range []string{"- One", "- Nested A", "- Nested B", "- Two"} {
		if !strings.Contains(md, want) {
			t.Errorf("expected list item %q in output:\n%s", want, md)
		}
	}

	// Nested items must be indented relative to their parent.
	nestedIndent := lineIndent(md, "Nested A")
	parentIndent := lineIndent(md, "One")
	if nestedIndent <= parentIndent {
		t.Errorf("expected nested item to be indented deeper than parent (nested=%d parent=%d):\n%s",
			nestedIndent, parentIndent, md)
	}
}

func TestConvertStrikethrough(t *testing.T) {
	html := `<p>This is <del>deleted</del> and <s>struck</s> text.</p>`

	md, err := Convert(html)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if !strings.Contains(md, "~~deleted~~") {
		t.Errorf("expected ~~deleted~~ in output:\n%s", md)
	}
	if !strings.Contains(md, "~~struck~~") {
		t.Errorf("expected ~~struck~~ in output:\n%s", md)
	}
}

// splitNonEmpty returns the non-blank, trimmed lines of s.
func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

// lineIndent returns the leading-space count of the first line containing sub,
// or -1 if no line contains it.
func lineIndent(s, sub string) int {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, sub) {
			return len(line) - len(strings.TrimLeft(line, " "))
		}
	}
	return -1
}

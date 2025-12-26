package htmlformat

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormat_MinifiedHTML(t *testing.T) {
	input := `<!DOCTYPE html><html><head><title>Test</title></head><body><div><p>Text</p></div></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Check that output is properly indented
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) < 8 {
		t.Errorf("Expected at least 8 lines, got %d", len(lines))
	}

	// Verify key elements are present and indented
	expected := []string{
		"<!DOCTYPE html>",
		"<html>",
		"  <head>",
		"    <title>",
		"      Test",
		"    </title>",
		"  </head>",
		"  <body>",
		"    <div>",
		"      <p>",
		"        Text",
		"      </p>",
		"    </div>",
		"  </body>",
		"</html>",
	}

	for i, exp := range expected {
		if i >= len(lines) {
			t.Errorf("Missing line %d: %q", i, exp)
			continue
		}
		if lines[i] != exp {
			t.Errorf("Line %d: got %q, want %q", i, lines[i], exp)
		}
	}
}

func TestFormat_NestedElements(t *testing.T) {
	input := `<div><ul><li>Item 1</li><li>Item 2</li></ul></div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := `<div>
  <ul>
    <li>
      Item 1
    </li>
    <li>
      Item 2
    </li>
  </ul>
</div>
`

	if result != expected {
		t.Errorf("Format() nested elements:\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_PreTagPreservation(t *testing.T) {
	input := `<pre>  Line 1
  Line 2
    Indented</pre>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// The pre tag content should be preserved exactly
	if !strings.Contains(result, "  Line 1\n  Line 2\n    Indented") {
		t.Errorf("Pre tag content not preserved:\n%s", result)
	}
}

func TestFormat_TextareaPreservation(t *testing.T) {
	input := `<textarea>  Spaces here
Multiple lines
  With   whitespace</textarea>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// The textarea content should be preserved exactly
	if !strings.Contains(result, "  Spaces here\nMultiple lines\n  With   whitespace") {
		t.Errorf("Textarea content not preserved:\n%s", result)
	}
}

func TestFormat_SelfClosingTags(t *testing.T) {
	input := `<div><img src="test.jpg"/><br/><hr/></div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify self-closing tags are on separate lines
	lines := strings.Split(strings.TrimSpace(result), "\n")
	expected := []string{
		"<div>",
		"  <img src=\"test.jpg\"/>",
		"  <br/>",
		"  <hr/>",
		"</div>",
	}

	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d:\n%s", len(expected), len(lines), result)
	}

	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("Line %d: got %q, want %q", i, lines[i], exp)
		}
	}
}

func TestFormat_CommentsAndDoctype(t *testing.T) {
	input := `<!DOCTYPE html><!-- Main content --><html><head><!-- Head section --><title>Test</title></head></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify doctype and comments are formatted
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}
	if !strings.Contains(result, "<!-- Main content -->") {
		t.Error("Missing main comment")
	}
	if !strings.Contains(result, "<!-- Head section -->") {
		t.Error("Missing head comment")
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if lines[0] != "<!DOCTYPE html>" {
		t.Errorf("First line should be DOCTYPE, got %q", lines[0])
	}
}

func TestFormat_TextNodeHandling(t *testing.T) {
	input := `<p>This is  some    text   with     spaces</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Whitespace should be collapsed
	expected := `<p>
  This is some text with spaces
</p>
`

	if result != expected {
		t.Errorf("Format() text handling:\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_EmptyElements(t *testing.T) {
	input := `<div></div><p></p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := `<div>
</div>
<p>
</p>
`

	if result != expected {
		t.Errorf("Format() empty elements:\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_NestedPreTag(t *testing.T) {
	input := `<div><pre>Code here
  With indentation</pre><p>Normal text</p></div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Pre content should be preserved, but surrounding elements should be formatted
	if !strings.Contains(result, "Code here\n  With indentation") {
		t.Error("Pre tag content not preserved")
	}
	if !strings.Contains(result, "  <pre>") {
		t.Error("Pre tag not indented properly")
	}
	if !strings.Contains(result, "  <p>") {
		t.Error("P tag not indented properly")
	}
}

func TestFormat_ComplexNesting(t *testing.T) {
	input := `<html><body><div class="container"><header><h1>Title</h1></header><main><article><h2>Article</h2><p>Content</p></article></main><footer>Footer</footer></div></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Check some key indentation levels
	lines := strings.Split(result, "\n")
	indentCount := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed != "" {
			indent := len(line) - len(trimmed)
			indentCount[trimmed] = indent
		}
	}

	// Verify proper nesting depth
	if indentCount["<html>"] != 0 {
		t.Errorf("html should have 0 indent, got %d", indentCount["<html>"])
	}
	if indentCount["<body>"] != 2 {
		t.Errorf("body should have 2 spaces indent, got %d", indentCount["<body>"])
	}
	if indentCount["<div class=\"container\">"] != 4 {
		t.Errorf("div should have 4 spaces indent, got %d", indentCount["<div class=\"container\">"])
	}
	if indentCount["<h1>"] != 8 {
		t.Errorf("h1 should have 8 spaces indent, got %d", indentCount["<h1>"])
	}
}

func TestFormat_AttributesPreserved(t *testing.T) {
	input := `<div id="main" class="container wide" data-value="test"><a href="http://example.com" target="_blank">Link</a></div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All attributes should be preserved
	if !strings.Contains(result, `id="main"`) {
		t.Error("id attribute not preserved")
	}
	if !strings.Contains(result, `class="container wide"`) {
		t.Error("class attribute not preserved")
	}
	if !strings.Contains(result, `data-value="test"`) {
		t.Error("data attribute not preserved")
	}
	if !strings.Contains(result, `href="http://example.com"`) {
		t.Error("href attribute not preserved")
	}
	if !strings.Contains(result, `target="_blank"`) {
		t.Error("target attribute not preserved")
	}
}

func TestFormat_MixedContent(t *testing.T) {
	input := `<p>Text before<strong>bold</strong>text after</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should have all text content
	if !strings.Contains(result, "Text before") {
		t.Error("Missing 'Text before'")
	}
	if !strings.Contains(result, "bold") {
		t.Error("Missing 'bold'")
	}
	if !strings.Contains(result, "text after") {
		t.Error("Missing 'text after'")
	}
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestFormat_ScriptTagPreservation(t *testing.T) {
	input := `<html><head><script>
function test() {
  if (true) {
    console.log("indented");
  }
}
</script></head></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// JavaScript should be preserved exactly
	if !strings.Contains(result, "function test() {\n  if (true) {\n    console.log(\"indented\");\n  }\n}") {
		t.Errorf("Script content not preserved:\n%s", result)
	}
}

func TestFormat_StyleTagPreservation(t *testing.T) {
	input := `<html><head><style>
body {
  margin: 0;
  padding: 0;
}
  .indent {
    color: red;
  }
</style></head></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// CSS should be preserved exactly
	if !strings.Contains(result, "body {\n  margin: 0;\n  padding: 0;\n}") {
		t.Errorf("Style content not preserved:\n%s", result)
	}
	if !strings.Contains(result, "  .indent {\n    color: red;\n  }") {
		t.Errorf("Style indentation not preserved:\n%s", result)
	}
}

func TestFormat_HTMLEntities(t *testing.T) {
	input := `<p>&amp; &lt; &gt; &nbsp; &quot; &#39; &copy;</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All entities should be preserved
	entities := []string{"&amp;", "&lt;", "&gt;", "&nbsp;", "&quot;", "&#39;", "&copy;"}
	for _, entity := range entities {
		if !strings.Contains(result, entity) {
			t.Errorf("Entity %s not preserved in output", entity)
		}
	}
}

func TestFormat_VoidElements(t *testing.T) {
	input := `<html><head><meta charset="utf-8"><link rel="stylesheet" href="style.css"></head><body><img src="test.jpg"><input type="text"><br><hr></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify void elements are on separate lines
	voidElements := []string{
		`<meta charset="utf-8">`,
		`<link rel="stylesheet" href="style.css">`,
		`<img src="test.jpg">`,
		`<input type="text">`,
		"<br>",
		"<hr>",
	}
	for _, elem := range voidElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Void element not found: %s", elem)
		}
	}
}

func TestFormat_DeeplyNestedStructure(t *testing.T) {
	// Build deeply nested structure
	var input strings.Builder
	input.WriteString("<html><body>")
	depth := 50
	for i := 0; i < depth; i++ {
		input.WriteString("<div>")
	}
	input.WriteString("Deep")
	for i := 0; i < depth; i++ {
		input.WriteString("</div>")
	}
	input.WriteString("</body></html>")

	result, err := Format(input.String())
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should handle deep nesting without error
	if !strings.Contains(result, "Deep") {
		t.Error("Deep text content lost")
	}

	// Check that we have proper indentation at various depths
	lines := strings.Split(result, "\n")
	foundDeep := false
	for _, line := range lines {
		if strings.Contains(line, "Deep") {
			foundDeep = true
			// Deep content should be heavily indented
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent < 50 { // Should be indented at least 50+ levels
				t.Errorf("Deep nesting not properly indented: %d spaces", indent)
			}
		}
	}
	if !foundDeep {
		t.Error("Could not find deep content in output")
	}
}

func TestFormat_MalformedHTML_UnclosedTags(t *testing.T) {
	// Tokenizer handles this - it will just stop when it hits EOF
	input := `<html><body><div><p>Text`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should still format what it can
	if !strings.Contains(result, "Text") {
		t.Error("Text content lost in malformed HTML")
	}
}

func TestFormat_InlineSVG(t *testing.T) {
	input := `<html><body><svg width="100" height="100"><circle cx="50" cy="50" r="40" fill="red"/></svg></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// SVG elements should be preserved
	if !strings.Contains(result, "<svg") {
		t.Error("SVG tag not found")
	}
	if !strings.Contains(result, "<circle") {
		t.Error("Circle tag not found")
	}
	if !strings.Contains(result, `fill="red"`) {
		t.Error("SVG attributes not preserved")
	}
}

func TestFormat_LongAttributeValues(t *testing.T) {
	longValue := strings.Repeat("x", 5000)
	input := fmt.Sprintf(`<img src="data:image/png;base64,%s" alt="test">`, longValue)
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Long attribute values should be preserved
	if !strings.Contains(result, longValue) {
		t.Error("Long attribute value was truncated or lost")
	}
}

func TestFormat_UnicodeAndEmoji(t *testing.T) {
	input := `<p>Hello 世界 🌍 مرحبا Привет</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Unicode and emoji should be preserved
	if !strings.Contains(result, "世界") {
		t.Error("Chinese characters not preserved")
	}
	if !strings.Contains(result, "🌍") {
		t.Error("Emoji not preserved")
	}
	if !strings.Contains(result, "مرحبا") {
		t.Error("Arabic text not preserved")
	}
	if !strings.Contains(result, "Привет") {
		t.Error("Cyrillic text not preserved")
	}
}

func TestFormat_CodeTagWhitespace(t *testing.T) {
	// Code tags are NOT in our preformatted list currently
	// This test documents current behavior - whitespace gets collapsed
	input := `<code>function    test()    {}</code>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Currently code tags collapse whitespace (not preformatted)
	// This is documented behavior - users should use <pre><code> for preservation
	if strings.Contains(result, "    test()    ") {
		t.Log("Note: code tag whitespace is being preserved (behavior may have changed)")
	}
}

func TestFormat_NestedScriptInBody(t *testing.T) {
	input := `<html><body><div><script>
var x = {
  nested: {
    deeply: true
  }
};
</script><p>Text</p></div></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Script should preserve its indentation
	if !strings.Contains(result, "var x = {\n  nested: {\n    deeply: true\n  }\n};") {
		t.Errorf("Nested script content not preserved:\n%s", result)
	}

	// But surrounding HTML should be formatted
	if !strings.Contains(result, "  <script>") || !strings.Contains(result, "  <p>") {
		t.Error("Surrounding HTML not properly formatted")
	}
}

func TestFormat_MultipleConsecutiveTextNodes(t *testing.T) {
	// This tests how we handle text, tag, text, tag patterns
	input := `<p>Start<strong>bold</strong>middle<em>italic</em>end</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All text should be present
	if !strings.Contains(result, "Start") {
		t.Error("Missing 'Start'")
	}
	if !strings.Contains(result, "bold") {
		t.Error("Missing 'bold'")
	}
	if !strings.Contains(result, "middle") {
		t.Error("Missing 'middle'")
	}
	if !strings.Contains(result, "italic") {
		t.Error("Missing 'italic'")
	}
	if !strings.Contains(result, "end") {
		t.Error("Missing 'end'")
	}
}

func TestFormat_EmptyScriptAndStyle(t *testing.T) {
	input := `<html><head><script></script><style></style></head></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Empty script and style tags should be present
	if !strings.Contains(result, "<script>") || !strings.Contains(result, "</script>") {
		t.Error("Empty script tags not preserved")
	}
	if !strings.Contains(result, "<style>") || !strings.Contains(result, "</style>") {
		t.Error("Empty style tags not preserved")
	}
}

func TestFormat_CommentsInUnusualPlaces(t *testing.T) {
	input := `<html><!-- before head --><head><title>Test</title></head><!-- between head and body --><body><!-- in body --><p>Text</p></body><!-- after body --></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All comments should be preserved
	comments := []string{
		"<!-- before head -->",
		"<!-- between head and body -->",
		"<!-- in body -->",
		"<!-- after body -->",
	}
	for _, comment := range comments {
		if !strings.Contains(result, comment) {
			t.Errorf("Comment not preserved: %s", comment)
		}
	}
}

func TestFormat_MixedSelfClosingStyles(t *testing.T) {
	// HTML5 allows <br> without slash, XML requires <br/>
	input := `<div><br><img src="a.jpg"/><hr><input type="text"/></div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All elements should be preserved (with or without slash)
	if !strings.Contains(result, "<br>") && !strings.Contains(result, "<br/>") {
		t.Error("br tag not found")
	}
	if !strings.Contains(result, "<hr>") && !strings.Contains(result, "<hr/>") {
		t.Error("hr tag not found")
	}
}

func TestFormat_SpecialCharactersInText(t *testing.T) {
	input := `<p>Special chars: < > & " ' @#$%^&*()</p>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Content should be preserved (though < and > might be encoded)
	if !strings.Contains(result, "@#$%^&*()") {
		t.Error("Special characters not preserved")
	}
}

func TestFormat_DataAttributes(t *testing.T) {
	input := `<div data-value="test" data-index="5" data-json='{"key":"value"}'>Content</div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All data attributes should be preserved
	if !strings.Contains(result, `data-value="test"`) {
		t.Error("data-value attribute not preserved")
	}
	if !strings.Contains(result, `data-index="5"`) {
		t.Error("data-index attribute not preserved")
	}
	if !strings.Contains(result, `data-json`) {
		t.Error("data-json attribute not preserved")
	}
}

func TestFormat_TemplateTag(t *testing.T) {
	input := `<template id="test"><div>Template content</div></template>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Template and its content should be preserved
	if !strings.Contains(result, "<template") {
		t.Error("Template tag not preserved")
	}
	if !strings.Contains(result, "Template content") {
		t.Error("Template content not preserved")
	}
}

func TestFormat_MixedLineEndings(t *testing.T) {
	// Test with mixed \n, \r\n, and \r line endings
	input := "<html>\r\n<body>\r<p>Test</p>\n</body>\r\n</html>"
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should handle mixed line endings gracefully
	if !strings.Contains(result, "Test") {
		t.Error("Content lost with mixed line endings")
	}
}

// ============================================================================
// ADDITIONAL EDGE CASES FROM RESEARCH
// ============================================================================

func TestFormat_CDATASection(t *testing.T) {
	// CDATA has no meaning in HTML (only in XML)
	// HTML parsers treat CDATA outside SVG/MathML as comments
	input := `<html><body><![CDATA[Some data here]]><p>Text</p></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// CDATA should be treated as a comment or preserved somehow
	if !strings.Contains(result, "Text") {
		t.Error("Text content lost when CDATA present")
	}
}

func TestFormat_UnquotedAttributes(t *testing.T) {
	// Unquoted attributes should be preserved
	input := `<div class=container data-value=123 id=main>Content</div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Attributes should be preserved (may be quoted or unquoted)
	if !strings.Contains(result, "class") {
		t.Error("class attribute not preserved")
	}
	if !strings.Contains(result, "data-value") {
		t.Error("data-value attribute not preserved")
	}
	if !strings.Contains(result, "id") {
		t.Error("id attribute not preserved")
	}
}

func TestFormat_UnquotedAttributeWithSlash(t *testing.T) {
	// Edge case: unquoted attribute ending with slash
	// The slash should not be interpreted as self-closing syntax
	input := `<img src=test.jpg/ alt=test>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should preserve the attribute value with slash
	if !strings.Contains(result, "test.jpg") || !strings.Contains(result, "alt") {
		t.Errorf("Unquoted attribute with slash not handled correctly:\n%s", result)
	}
}

func TestFormat_AllVoidElements(t *testing.T) {
	// Test all HTML5 void elements
	input := `<html><head>
<meta charset="utf-8">
<link rel="stylesheet" href="style.css">
<base href="/">
</head><body>
<img src="test.jpg">
<input type="text">
<br>
<hr>
<area shape="rect" coords="0,0,100,100" href="/">
<col span="2">
<embed src="video.mp4">
<param name="autoplay" value="true">
<source src="audio.mp3" type="audio/mpeg">
<track src="subtitles.vtt" kind="subtitles">
<wbr>
</body></html>`

	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// All void elements should be present
	voidElements := []string{
		"<meta", "<link", "<base", "<img", "<input",
		"<br", "<hr", "<area", "<col", "<embed",
		"<param", "<source", "<track", "<wbr",
	}
	for _, elem := range voidElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Void element not found: %s", elem)
		}
	}
}

func TestFormat_PreTagWithLeadingNewline(t *testing.T) {
	// First newline after <pre> should be ignored per HTML spec
	input := "<pre>\nLine 1\nLine 2</pre>"
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// The <pre> content should be preserved
	// Note: golang.org/x/net/html may or may not strip the leading newline
	if !strings.Contains(result, "Line 1") {
		t.Error("Pre content not preserved")
	}
}

func TestFormat_NonVoidSelfClosingTag(t *testing.T) {
	// Invalid HTML: non-void elements with self-closing syntax
	// Should be handled gracefully (tokenizer will treat it as start tag)
	input := `<div/><span/><p/>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should not crash - exact output depends on tokenizer behavior
	if len(result) == 0 {
		t.Error("Empty result from non-void self-closing tags")
	}
}

func TestFormat_SVGSelfClosing(t *testing.T) {
	// SVG elements can use self-closing syntax
	input := `<svg><circle cx="50" cy="50" r="40"/><rect x="0" y="0" width="100" height="100"/></svg>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// SVG elements should be preserved
	if !strings.Contains(result, "circle") {
		t.Error("SVG circle element not found")
	}
	if !strings.Contains(result, "rect") {
		t.Error("SVG rect element not found")
	}
}

func TestFormat_MathML(t *testing.T) {
	// MathML inline in HTML5
	input := `<html><body><math><mrow><mi>x</mi><mo>+</mo><mn>1</mn></mrow></math></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// MathML elements should be preserved
	if !strings.Contains(result, "<math") {
		t.Error("MathML math element not found")
	}
	if !strings.Contains(result, "<mi>") {
		t.Error("MathML mi element not found")
	}
}

func TestFormat_ScriptWithLeadingTrailingWhitespace(t *testing.T) {
	// Script tags with whitespace before/after content
	input := `<script>

  var x = 1;
  
</script>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Whitespace in script should be preserved exactly
	if !strings.Contains(result, "var x = 1;") {
		t.Error("Script content not preserved")
	}
}

func TestFormat_StyleWithComments(t *testing.T) {
	// CSS with comments
	input := `<style>
/* Comment */
body { margin: 0; }
/* Another comment */
</style>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// CSS comments should be preserved
	if !strings.Contains(result, "/* Comment */") {
		t.Error("CSS comment not preserved")
	}
	if !strings.Contains(result, "body { margin: 0; }") {
		t.Error("CSS rule not preserved")
	}
}

func TestFormat_BooleanAttributes(t *testing.T) {
	// Boolean attributes (no value needed)
	input := `<input type="checkbox" checked disabled readonly>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Boolean attributes should be preserved
	if !strings.Contains(result, "checked") {
		t.Error("checked attribute not found")
	}
	if !strings.Contains(result, "disabled") {
		t.Error("disabled attribute not found")
	}
}

func TestFormat_HTML5CustomElements(t *testing.T) {
	// Web components with custom element names
	input := `<my-component><custom-header>Title</custom-header><x-content>Body</x-content></my-component>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Custom elements should be preserved
	if !strings.Contains(result, "my-component") {
		t.Error("my-component not found")
	}
	if !strings.Contains(result, "custom-header") {
		t.Error("custom-header not found")
	}
}

func TestFormat_MultipleClasses(t *testing.T) {
	// Multiple classes with various spacing
	input := `<div class="class1  class2   class3">Content</div>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Class attribute should be preserved (spaces may be normalized)
	if !strings.Contains(result, "class") {
		t.Error("class attribute not found")
	}
}

func TestFormat_ScriptType(t *testing.T) {
	// Script with type attribute
	input := `<script type="application/json">{"key": "value"}</script>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// JSON in script should be preserved
	if !strings.Contains(result, `{"key": "value"}`) {
		t.Error("JSON in script not preserved")
	}
}

func TestFormat_NoscriptTag(t *testing.T) {
	// Noscript tag content
	input := `<noscript><p>Please enable JavaScript</p></noscript>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Noscript content should be formatted
	if !strings.Contains(result, "<noscript>") {
		t.Error("noscript tag not found")
	}
	if !strings.Contains(result, "Please enable JavaScript") {
		t.Error("noscript content not found")
	}
}

func TestFormat_IndentAfterRawTag(t *testing.T) {
	// This test catches the bug where indentation drifts after raw tags (style, script, pre, textarea)
	// The bug: opening a raw tag increments indentLevel, but closing it doesn't decrement
	input := `<html><head><meta charset="utf-8"><style>body{margin:0;}</style><title>Test</title></head><body><p>Content</p></body></html>`
	result, err := Format(input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := `<html>
  <head>
    <meta charset="utf-8">
    <style>
body{margin:0;}</style>
    <title>
      Test
    </title>
  </head>
  <body>
    <p>
      Content
    </p>
  </body>
</html>
`

	if result != expected {
		t.Errorf("Format() indentation after raw tag incorrect:\ngot:\n%s\nwant:\n%s", result, expected)

		// Show line-by-line comparison
		gotLines := strings.Split(result, "\n")
		wantLines := strings.Split(expected, "\n")
		for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
			got := ""
			want := ""
			if i < len(gotLines) {
				got = gotLines[i]
			}
			if i < len(wantLines) {
				want = wantLines[i]
			}
			if got != want {
				t.Errorf("Line %d differs:\n  got:  %q\n  want: %q", i, got, want)
			}
		}
	}
}

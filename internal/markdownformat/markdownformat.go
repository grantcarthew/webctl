// Package markdownformat converts HTML to Markdown.
//
// It wraps html-to-markdown/v2 with the same plugin set (base, commonmark,
// table, strikethrough) as the snag tool so that webctl and snag produce
// identical Markdown for the same input.
package markdownformat

import (
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

// conv is constructed once and reused. The converter is safe for repeated
// ConvertString calls and holds no per-conversion state.
var conv = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(),
		table.NewTablePlugin(),
		strikethrough.NewStrikethroughPlugin(),
	),
)

// Convert converts an HTML string to Markdown.
func Convert(html string) (string, error) {
	return conv.ConvertString(html)
}

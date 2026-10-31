package ui

import (
	"bytes"
	"html"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var (
	maxHeadingLevel = 6
)

// heading is a heading extracted from a markdown document.
type heading struct {
	level        int    // 1-6, corresponding to # through ######.
	text         string // The heading text, without the # prefix.
	sourceLine   int    // Zero-based line number in the source markdown.
	renderedLine int    // Zero-based line in the rendered output; -1 if unmapped.
}

// outlineModel manages the outline sidebar state.
type outlineModel struct {
	headings []heading
	selected int            // The selected heading when the outline is focused.
	current  int            // The heading at the current scroll position.
	width    int            // Sidebar width.
	height   int            // Available height.
	visible  bool           // Whether the outline can be shown at this width.
	focused  bool           // Whether the outline has keyboard focus.
	viewport viewport.Model // Scrollable viewport for long outlines.
}

// nodeText extracts the plain text of a heading.
func nodeText(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch c := child.(type) {
		case *ast.Text:
			b.Write(c.Segment.Value(source))
			if c.SoftLineBreak() || c.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(c.Value)
		}

		return ast.WalkContinue, nil
	})

	text := normalizeHeadingText(b.String())
	return text
}

// normalizeHeadingText normalizes entities and whitespace for comparisons.
func normalizeHeadingText(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// removeHeadingMarkers removes the # markers glamour adds to rendered headings.
func removeHeadingMarkers(s string) string {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] == '#' {
		i++
	}
	// valid marker length
	if i <= 0 || i > maxHeadingLevel || i >= len(s) {
		return s
	}
	if s[i] == ' ' || s[i] == '\t' {
		// heading text
		return strings.TrimSpace(s[i:])
	}
	return s
}

// headingLineMatches reports whether a rendered line belongs to the given heading text.
func headingLineMatches(rendered, target string) bool {
	rendered = normalizeHeadingText(removeHeadingMarkers(rendered))
	target = normalizeHeadingText(target)
	if rendered == "" || target == "" {
		return false
	}

	// to same format
	renderedLower := strings.ToLower(rendered)
	targetLower := strings.ToLower(target)

	// text
	if renderedLower == targetLower {
		return true
	}
	// Long text cut into multirendereds in rendered
	// like: # This is A Long Text
	// tobe: This is A Long
	//	 Text
	// compare the first 8 char
	if len(renderedLower) >= 8 && strings.HasPrefix(targetLower, renderedLower) {
		return true
	}
	// Url append text
	// like: # [Docs](https://example.com)
	// tobe: Docs https://example.com
	if strings.HasPrefix(renderedLower, targetLower) {
		return true
	}
	return false
}

// parseHeadings extracts headings from raw markdown.
func parseHeadings(source []byte) []heading {
	// use GFM extension
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := md.Parser().Parse(text.NewReader(source))

	var headings []heading
	// traverse the heading text recurisively.
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// only heading node
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		// extract heading text
		text := nodeText(h, source)
		if text != "" {
			// setext
			// like: A Setext
			//	 Line
			//	 Another Line
			//	 ===
			// has 3 line
			line := 0
			if lines := h.Lines(); lines.Len() > 0 {
				// count heading real line
				line = bytes.Count(source[:lines.At(0).Start] /*the start of this heading line*/, []byte("\n"))
			}
			headings = append(headings, heading{
				level:        h.Level,
				text:         text,
				sourceLine:   line,
				renderedLine: -1,
			})
		}

		return ast.WalkSkipChildren, nil
	})

	return headings
}

// trimLineNumber removes the line-number gutter that the pager adds.
// use this when line numbers are enabled.
func trimLineNumber(s string) string {
	i := 0
	// start whitespace
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	// line-number
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return s
	}
	if i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	// pure content
	return s[i:]
}

// mapHeadingsToRenderedLines is best-effort filling each heading's renderedLine by matching it against the rendered content in rendered document order.
// Headings that cannot be mapped keep renderedLine = -1.
// param rendered is rendered markdown content
//
// this function may match the same string in main text but the certain rendered heading
func mapHeadingsToRenderedLines(headings []heading, rendered string, showLineNumbers bool) []heading {
	// clear renderedLine field
	for i := range headings {
		headings[i].renderedLine = -1
	}
	if rendered == "" || len(headings) == 0 {
		return headings
	}

	lines := strings.Split(rendered, "\n")
	// search cursor only move back
	searchStart := 0

	for i := range headings {
		if headings[i].text == "" {
			continue
		}

		for j := searchStart; j < len(lines); j++ {
			// plain text
			plain := ansi.Strip(lines[j])
			if showLineNumbers {
				plain = trimLineNumber(plain)
			}
			if headingLineMatches(plain, headings[i].text) {
				headings[i].renderedLine = j
				searchStart = j + 1
				break
			}
		}
	}

	return headings
}

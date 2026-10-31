package ui

import (
	"bytes"
	"html"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// const
const (
	maxHeadingLevel = 6

	outlineMinWidth     = 20
	outlineMaxWidth     = 40
	outlineWidthPercent = 25
	minTerminalWidth    = 80
)

// style
var (
	outlineTitleStyle    = lipgloss.NewStyle().Bold(true)
	outlineNormalStyle   = lipgloss.NewStyle()
	outlineCurrentStyle  = lipgloss.NewStyle().Bold(true)
	outlineSelectedStyle = lipgloss.NewStyle().Reverse(true)
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

func newOutlineModel() outlineModel {
	return outlineModel{
		viewport: viewport.New(),
		visible:  false,
		focused:  false,
	}
}

func calculateOutlineWidth(termWidth int) int {
	if termWidth < minTerminalWidth {
		return 0
	}
	width := termWidth * outlineWidthPercent / 100
	return min(max(width, outlineMinWidth), outlineMaxWidth)
}

// setContent parses the given markdown and resets outline state.
func (m *outlineModel) setContent(markdown string) {
	m.headings = parseHeadings([]byte(markdown))
	m.selected = 0
	m.current = 0
	m.updateViewport()
}

// setSize updates the outline dimensions from the terminal width. The pager
// controls whether the outline is visible.
func (m *outlineModel) setSize(termWidth, height int) {
	m.width = calculateOutlineWidth(termWidth)
	m.height = height
	m.viewport.SetWidth(m.width)
	// One row is reserved for the outline title.
	m.viewport.SetHeight(max(0, height-1))
	m.updateViewport()
}

// renderHeadingLine renders a single heading entry with
// indentation and highlight for the current scroll position and the selected entry.
func (m *outlineModel) renderHeadingLine(index int, h heading) string {
	indent := strings.Repeat("  ", max(h.level-1, 0))

	prefix := "  "
	if index == m.current {
		prefix = "> "
	}

	line := indent + prefix + h.text
	if m.width > 0 {
		line = truncate.StringWithTail(line, uint(m.width), "…")
	}

	switch {
	case m.focused && index == m.selected:
		return outlineSelectedStyle.Width(m.width).Render(line)
	case index == m.current:
		return outlineCurrentStyle.Width(m.width).Render(line)
	default:
		return outlineNormalStyle.Width(m.width).Render(line)
	}
}

// updateViewport re-renders all headings into the viewport content.
func (m *outlineModel) updateViewport() {
	if m.width == 0 || len(m.headings) == 0 {
		m.viewport.SetContent("")
		return
	}

	var b strings.Builder
	for i, h := range m.headings {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.renderHeadingLine(i, h))
	}
	m.viewport.SetContent(b.String())
}

// moveCursorUp moves the outline selection up one row.
func (m *outlineModel) moveCursorUp() {
	if m.selected > 0 {
		m.selected--
		m.ensureCursorVisible()
		m.updateViewport()
	}
}

// moveCursorDown moves the outline selection down one row.
func (m *outlineModel) moveCursorDown() {
	if m.selected < len(m.headings)-1 {
		m.selected++
		m.ensureCursorVisible()
		m.updateViewport()
	}
}

// ensureCursorVisible scrolls the outline viewport so the selected heading is visible.
func (m *outlineModel) ensureCursorVisible() {
	m.ensureIndexVisible(m.selected)
}

// ensureCurrentVisible scrolls the outline viewport so the current heading is visible.
func (m *outlineModel) ensureCurrentVisible() {
	m.ensureIndexVisible(m.current)
}

// ensureIndexVisible scrolls the outline viewport so the index of text line is visible
func (m *outlineModel) ensureIndexVisible(index int) {
	if index < 0 || index >= len(m.headings) || m.viewport.Height() <= 0 {
		return
	}

	// view field on screen
	top := m.viewport.YOffset()
	bottom := top + m.viewport.Height() - 1
	if index < top {
		m.viewport.SetYOffset(index)
		return
	}
	if index > bottom {
		m.viewport.SetYOffset(index - m.viewport.Height() + 1)
	}
}

// updateCurrent updates the heading that is highlighted for the document's
// current scroll position.
func (m *outlineModel) updateCurrent(yOffset int) {
	if len(m.headings) == 0 {
		m.current = 0
		return
	}

	current := 0
	for i, h := range m.headings {
		if h.renderedLine >= 0 && h.renderedLine <= yOffset {
			current = i
		}
	}

	if current != m.current {
		m.current = current
		m.ensureCurrentVisible()
		m.updateViewport()
	}
}

// View renders the outline sidebar.
func (m outlineModel) View() string {
	if m.width <= 0 {
		return ""
	}

	title := outlineTitleStyle.Width(m.width).Render("OUTLINE")
	if len(m.headings) == 0 {
		return title + "\n" + outlineNormalStyle.Width(m.width).Render("  No headings")
	}
	return title + "\n" + m.viewport.View()
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
// NOTE: before adding this outline-sidebar to glow, frozen this implement.
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

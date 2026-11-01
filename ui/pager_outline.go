package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/glow/v3/utils"
	"github.com/muesli/reflow/ansi"
)

func (m pagerModel) isMarkdownFile() bool {
	return utils.IsMarkdownFile(m.currentDocument.Note)
}

func clampIndex(index, length int) int {
	if length <= 0 || index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

// handleOutlineKey handles the outline-specific keys.
// It reports whether the key was consumed so the pager's normal key handling can continue otherwise.
func (m *pagerModel) handleOutlineKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "o":
		if !m.isMarkdownFile() {
			return nil, true
		}

		m.showOutline = !m.showOutline

		if m.showOutline {
			m.outline.setContent(m.currentDocument.Body)
			m.outline.updateCurrent(m.viewport.YOffset())
		} else {
			m.outline.focused = false
		}

		m.setSize(m.common.width, m.common.height)

		if m.showOutline && m.outline.visible {
			m.outline.focused = true
			m.outline.selected = m.outline.current
		}

		m.outline.updateViewport()
		return renderWithGlamour(*m, m.currentDocument.Body), true
	}

	if msg.String() == "tab" {
		if !m.showOutline || !m.outline.visible {
			return nil, false
		}
		m.outline.focused = !m.outline.focused
		if m.outline.focused {
			m.outline.selected = m.outline.current
		}
		m.outline.updateViewport()
		return nil, true
	}

	if !m.showOutline || !m.outline.visible || !m.outline.focused {
		return nil, false
	}

	switch msg.String() {
	case "j", "down":
		m.outline.moveCursorDown()
		return nil, true
	case "k", "up":
		m.outline.moveCursorUp()
		return nil, true
	case "enter":
		if len(m.outline.headings) == 0 {
			return nil, true
		}
		m.jumpToHeading(m.outline.selected)
		return nil, true
	}

	return nil, false
}

// handleOutlineContentRendered refreshes heading positions after a render.
func (m *pagerModel) handleOutlineContentRendered(rendered string) {
	if !m.isMarkdownFile() || !m.showOutline {
		return
	}

	selected := m.outline.selected
	current := m.outline.current

	m.outline.setContent(m.currentDocument.Body)
	// selected current must in outline sidebar content
	m.outline.selected = clampIndex(selected, len(m.outline.headings))
	m.outline.current = clampIndex(current, len(m.outline.headings))

	m.outline.headings = mapHeadingsToRenderedLines(
		m.outline.headings,
		rendered,
		m.common.cfg.ShowLineNumbers,
	)

	m.outline.updateCurrent(m.viewport.YOffset())
	m.outline.updateViewport()
}

func (m *pagerModel) jumpToHeading(index int) {
	if index < 0 || index >= len(m.outline.headings) {
		return
	}

	h := m.outline.headings[index]
	if h.renderedLine < 0 {
		m.jumpToSourceLine(h.sourceLine)
	} else {
		m.setViewportTarget(h.renderedLine)
	}

	m.outline.current = index
	m.outline.selected = index
	m.outline.updateViewport()
}

func (m *pagerModel) jumpToSourceLine(sourceLine int) {
	totalSource := strings.Count(m.currentDocument.Body, "\n") + 1
	totalRendered := m.viewport.TotalLineCount()
	if totalSource <= 0 || totalRendered <= 0 {
		return
	}

	m.setViewportTarget(sourceLine * totalRendered / totalSource)
}

func (m *pagerModel) setViewportTarget(line int) {
	line = max(0, line-2)
	maxOffset := max(0, m.viewport.TotalLineCount()-m.viewport.Height())
	if line > maxOffset {
		line = maxOffset
	}
	m.viewport.SetYOffset(line)
}

// joinContentAndOutline joins the main content and the sidebar line by line,
// padding both to their configured widths.
//
// NOTE: Assumes each line of content and outline occupies one visual row.
//	 Headings using the text sizing protocol may span multiple rows on supporting
// 	 terminals, causing sidebar misalignment.
//	 Known issue in glow v3.
//
// Verified on
// kitty (supports the text sizing protocol): the sidebar is misaligned.
// Konsole (does not support it): renders correctly.
func (m pagerModel) joinContentAndOutline(content, outline string) string {
	contentLines := strings.Split(content, "\n")
	outlineLines := strings.Split(outline, "\n")
	lineCount := max(len(contentLines), len(outlineLines))

	contentWidth := max(0, m.viewport.Width())
	outlineWidth := max(0, m.outline.width)

	var b strings.Builder
	for i := 0; i < lineCount; i++ {
		var contentLine, outlineLine string
		if i < len(contentLines) {
			contentLine = contentLines[i]
		}
		if i < len(outlineLines) {
			outlineLine = outlineLines[i]
		}

		if width := ansi.PrintableRuneWidth(contentLine); width < contentWidth {
			contentLine += strings.Repeat(" ", contentWidth-width)
		}
		if width := ansi.PrintableRuneWidth(outlineLine); width < outlineWidth {
			outlineLine += strings.Repeat(" ", outlineWidth-width)
		}

		b.WriteString(contentLine)
		b.WriteString(outlineLine)
		if i < lineCount-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

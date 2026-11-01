package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func newOutlineTestPager() pagerModel {
	common := &commonModel{
		cfg: Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
		},
		styles: newStyles(true),
		width:  100,
		height: 24,
	}

	m := newPagerModel(common)
	// # One
	//
	// ## Two
	//
	m.currentDocument = markdown{
		Note: "test.md",
		Body: "# One\n\n## Two\n",
	}
	m.setSize(100, 24)
	return m
}

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}

func TestPagerOutlineToggle(t *testing.T) {
	m := newOutlineTestPager()

	cmd, handled := m.handleOutlineKey(keyPress("o"))
	if !handled {
		t.Fatal("expected o to be handled")
	}
	if cmd == nil {
		t.Fatal("expected a render command")
	}
	if !m.showOutline {
		t.Error("expected showOutline=true")
	}
	if !m.outline.visible {
		t.Error("expected outline.visible=true")
	}
	if !m.outline.focused {
		t.Error("expected outline.focused=true")
	}
	if m.viewport.Width() != 75 {
		t.Errorf("expected content width 75, got %d", m.viewport.Width())
	}

	cmd, handled = m.handleOutlineKey(keyPress("o"))
	if !handled || cmd == nil {
		t.Fatal("expected o to close the outline and request a render")
	}
	if m.showOutline {
		t.Error("expected showOutline=false")
	}
	if m.outline.focused {
		t.Error("expected outline.focused=false")
	}
	if m.viewport.Width() != 100 {
		t.Errorf("expected content width 100, got %d", m.viewport.Width())
	}
}

func TestPagerOutlineToggleNonMarkdown(t *testing.T) {
	m := newOutlineTestPager()
	m.currentDocument.Note = "main.go"

	_, handled := m.handleOutlineKey(keyPress("o"))
	if !handled {
		t.Fatal("expected o to be handled")
	}
	if m.showOutline {
		t.Error("outline should not open for non-markdown files")
	}
}

func TestPagerOutlineNavigate(t *testing.T) {
	m := newOutlineTestPager()
	m.handleOutlineKey(keyPress("o"))

	m.handleOutlineKey(keyPress("j"))
	if m.outline.selected != 1 {
		t.Errorf("expected selected=1, got %d", m.outline.selected)
	}

	m.handleOutlineKey(keyPress("k"))
	if m.outline.selected != 0 {
		t.Errorf("expected selected=0, got %d", m.outline.selected)
	}
}

func TestPagerOutlineTabFocus(t *testing.T) {
	m := newOutlineTestPager()
	m.handleOutlineKey(keyPress("o"))

	m.handleOutlineKey(keyPress("tab"))
	if m.outline.focused {
		t.Error("expected outline to lose focus after tab")
	}

	m.handleOutlineKey(keyPress("tab"))
	if !m.outline.focused {
		t.Error("expected outline to regain focus after tab")
	}
}

func TestPagerOutlineEnterJumpsWithoutClosing(t *testing.T) {
	m := newOutlineTestPager()
	m.handleOutlineKey(keyPress("o"))

	// Intro
	//
	// body
	//
	// ## One
	//
	// body
	//
	// ## Two
	//
	rendered := "Intro\n\nbody\n\n## One\n\nbody\n\n## Two\n"
	m.setContent(rendered)
	m.handleOutlineContentRendered(rendered)

	m.handleOutlineKey(keyPress("j"))
	cmd, handled := m.handleOutlineKey(keyPress("enter"))
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	if cmd != nil {
		t.Error("expected enter to jump without requesting a render")
	}
	if !m.showOutline {
		t.Error("expected outline to stay visible")
	}
	if !m.outline.focused {
		t.Error("expected outline to stay focused")
	}
	if m.outline.current != 1 {
		t.Errorf("expected current=1 after jump, got %d", m.outline.current)
	}
	if m.outline.selected != 1 {
		t.Errorf("expected selected=1 after jump, got %d", m.outline.selected)
	}
}

func TestPagerOutlineNoHeadings(t *testing.T) {
	m := newOutlineTestPager()
	m.currentDocument.Body = "plain text\n"
	m.setContent("plain text\n")

	m.handleOutlineKey(keyPress("o"))
	if !strings.Contains(m.outline.View(), "No headings") {
		t.Errorf("expected empty-state message, got %q", m.outline.View())
	}

	m.handleOutlineKey(keyPress("j"))
	m.handleOutlineKey(keyPress("enter"))
	if !m.showOutline {
		t.Error("enter should not close the outline without headings")
	}
}

func TestPagerOutlineView(t *testing.T) {
	m := newOutlineTestPager()
	m.handleOutlineKey(keyPress("o"))

	// Intro
	//
	// ## One
	//
	// body
	//
	// ## Two
	//
	rendered := "Intro\n\n## One\n\nbody\n\n## Two\n"
	m.setContent(rendered)
	m.handleOutlineContentRendered(rendered)

	view := m.View()
	if !strings.Contains(view, "OUTLINE") {
		t.Errorf("expected sidebar in view, got %q", view)
	}
	if !strings.Contains(view, "One") || !strings.Contains(view, "Two") {
		t.Errorf("expected headings in view, got %q", view)
	}

	lines := strings.Split(view, "\n")
	if len(lines) != m.common.height {
		t.Errorf("view has %d lines, expected terminal height %d", len(lines), m.common.height)
	}
	for i, line := range lines {
		if width := lipgloss.Width(line); width > m.common.width {
			t.Errorf("line %d width %d exceeds terminal width %d: %q", i, width, m.common.width, line)
		}
	}
}

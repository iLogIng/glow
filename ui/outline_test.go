package ui

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestParseHeadings(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []heading
	}{
		{
			// # ATX
			//
			// Setext
			// ======
			//
			// ## Second
			//
			name:   "atx and setext headings",
			source: "# ATX\n\nSetext\n======\n\n## Second\n",
			want: []heading{
				{level: 1, text: "ATX", sourceLine: 0, renderedLine: -1},
				{level: 1, text: "Setext", sourceLine: 2, renderedLine: -1},
				{level: 2, text: "Second", sourceLine: 5, renderedLine: -1},
			},
		},
		{
			// # Read
			//
			// ```md
			// # not a heading
			// ```
			//
			//     # also not a heading
			//
			// ## Real Two
			//
			name: "fenced and indented code blocks",
			source: "# Real\n\n```md\n# not a heading\n```\n\n" +
				"    # also not a heading\n\n## Real Two\n",
			want: []heading{
				{level: 1, text: "Real", sourceLine: 0, renderedLine: -1},
				{level: 2, text: "Real Two", sourceLine: 8, renderedLine: -1},
			},
		},
		{
			// # `code` &amp; ~~strike~~ and [link](https://example.com)
			//
			name:   "inline markup and html entity",
			source: "# `code` &amp; ~~strike~~ and [link](https://example.com)\n",
			want: []heading{
				{level: 1, text: "code & strike and link", sourceLine: 0, renderedLine: -1},
			},
		},
		{
			// # ![alt](img.png)
			//
			//
			// # `only code`
			//
			name:   "image alt text and code span",
			source: "# ![alt](img.png)\n\n# `only code`\n",
			want: []heading{
				{level: 1, text: "alt", sourceLine: 0, renderedLine: -1},
				{level: 1, text: "only code", sourceLine: 2, renderedLine: -1},
			},
		},
		{
			// #
			//
			// ## Real
			name:   "empty heading is ignored",
			source: "#\n\n## Real\n",
			want: []heading{
				{level: 2, text: "Real", sourceLine: 2, renderedLine: -1},
			},
		},
		{
			//
			name:   "empty document",
			source: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeadings([]byte(tt.source))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseHeadings() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMapHeadingsToRenderedLines(t *testing.T) {
	tests := []struct {
		name            string
		headings        []heading
		rendered        string
		showLineNumbers bool
		want            []int
	}{
		{
			name:     "simple heading",
			headings: []heading{{text: "Installation"}},
			rendered: "Intro\n\n## Installation\n\nbody\n",
			want:     []int{2},
		},
		{
			name:     "body mentions heading text before the heading",
			headings: []heading{{text: "Installation"}},
			rendered: "We talk about Installation later.\n\n## Installation\n",
			want:     []int{2},
		},
		{
			name:     "wrapped heading",
			headings: []heading{{text: "a very long heading that will wrap when width is small"}},
			rendered: "## a very long heading that will wrap\nwhen width is small\n",
			want:     []int{0},
		},
		{
			name:     "inline code non-breaking space",
			headings: []heading{{text: "code heading"}},
			rendered: "## code\u00a0heading\n",
			want:     []int{0},
		},
		{
			name:     "html entity",
			headings: []heading{{text: "A & B"}},
			rendered: "## A &amp; B\n",
			want:     []int{0},
		},
		{
			name:     "link url suffix",
			headings: []heading{{text: "docs"}},
			rendered: "## docs https://example.com\n",
			want:     []int{0},
		},
		{
			name:            "line numbers",
			headings:        []heading{{text: "Installation"}},
			rendered:        "   1 ## Installation\n   2 body\n",
			showLineNumbers: true,
			want:            []int{0},
		},
		{
			name:     "duplicate headings",
			headings: []heading{{text: "Installation"}, {text: "Installation"}},
			rendered: "## Installation\n\nbody\n\n## Installation\n",
			want:     []int{0, 4},
		},
		{
			name:     "ansi styled heading",
			headings: []heading{{text: "Installation"}},
			rendered: "\x1b[1;38;5;39m## Installation\x1b[0m\n",
			want:     []int{0},
		},
		{
			name:     "unmapped heading",
			headings: []heading{{text: "Installation"}},
			rendered: "## Other\n",
			want:     []int{-1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapHeadingsToRenderedLines(tt.headings, tt.rendered, tt.showLineNumbers)
			lines := make([]int, len(got))
			for i := range got {
				lines[i] = got[i].renderedLine
			}
			if !reflect.DeepEqual(lines, tt.want) {
				t.Fatalf("rendered lines = %v, want %v", lines, tt.want)
			}
		})
	}
}

func TestNewOutlineModel(t *testing.T) {
	m := newOutlineModel()

	if m.visible {
		t.Error("new outline should not be visible")
	}
	if m.focused {
		t.Error("new outline should not be focused")
	}
	if len(m.headings) != 0 {
		t.Errorf("new outline should have no headings, got: %d", len(m.headings))
	}
	if m.width != 0 || m.height != 0 {
		t.Errorf("new outline should have zero size, got: %dx%d", m.width, m.height)
	}
}

func TestOutlineSetContent(t *testing.T) {
	m := newOutlineModel()
	m.setContent("# One\n\n## Two\n")

	if len(m.headings) != 2 {
		t.Fatalf("expected 2 headings, got: %d", len(m.headings))
	}
	if m.selected != 0 || m.current != 0 {
		t.Errorf("expected selected/current reset to 0, got: %d/%d", m.selected, m.current)
	}
	if m.headings[0].text != "One" || m.headings[1].text != "Two" {
		t.Errorf("unexpected heading text: %#v", m.headings)
	}
}

func TestOutlineSetSize(t *testing.T) {
	m := newOutlineModel()
	m.setSize(100, 10)

	if m.width != 25 {
		t.Errorf("expected width 25, got: %d", m.width)
	}
	if m.height != 10 {
		t.Errorf("expected height 10, got: %d", m.height)
	}
	if m.viewport.Width() != 24 {
		t.Errorf("expected viewport width 24 (one column for border), got: %d", m.viewport.Width())
	}
	if m.viewport.Height() != 9 {
		t.Errorf("expected viewport height 9 (one row for title), got: %d", m.viewport.Height())
	}

	m.setSize(40, 10)
	if m.width != 0 {
		t.Errorf("expected zero width on narrow terminal, got: %d", m.width)
	}
}

func TestOutlineMoveCursor(t *testing.T) {
	m := newOutlineModel()
	m.setContent("# One\n\n## Two\n")
	m.setSize(100, 10)

	m.moveCursorDown()
	if m.selected != 1 {
		t.Errorf("expected selected 1, got: %d", m.selected)
	}
	m.moveCursorDown()
	if m.selected != 1 {
		t.Errorf("expected selected clamped at 1, got: %d", m.selected)
	}
	m.moveCursorUp()
	if m.selected != 0 {
		t.Errorf("expected selected 0, got: %d", m.selected)
	}
	m.moveCursorUp()
	if m.selected != 0 {
		t.Errorf("expected selected clamped at 0, got: %d", m.selected)
	}
}

func TestOutlineUpdateCurrent(t *testing.T) {
	m := newOutlineModel()
	m.headings = []heading{
		{renderedLine: 0},
		{renderedLine: 10},
		{renderedLine: -1},
	}
	m.setSize(100, 10)

	m.updateCurrent(5)
	if m.current != 0 {
		t.Errorf("expected current 0, got: %d", m.current)
	}

	m.updateCurrent(10)
	if m.current != 1 {
		t.Errorf("expected current 1, got: %d", m.current)
	}

	m.updateCurrent(0)
	if m.current != 0 {
		t.Errorf("expected current 0, got: %d", m.current)
	}
}

func TestOutlineView(t *testing.T) {
	m := newOutlineModel()
	m.setContent("# One\n\n## Two\n")
	m.setSize(100, 10)

	view := m.View()
	if !strings.Contains(view, "OUTLINE") {
		t.Errorf("expected OUTLINE title, got: %q", view)
	}
	if !strings.Contains(view, "One") || !strings.Contains(view, "Two") {
		t.Errorf("expected headings in view, got: %q", view)
	}
}

func TestOutlineLongHeadingTruncation(t *testing.T) {
	m := newOutlineModel()
	m.setContent("# a very long heading that should be truncated\n")
	m.setSize(100, 10)

	line := m.renderHeadingLine(0, m.headings[0])
	if width := lipgloss.Width(line); width > m.innerWidth() {
		t.Errorf("heading line width %d exceeds inner width %d: %q", width, m.innerWidth(), line)
	}
	if !strings.Contains(line, "…") {
		t.Errorf("expected truncated heading to contain ellipsis, got: %q", line)
	}
}

func TestOutlineEmptyView(t *testing.T) {
	m := newOutlineModel()
	m.setSize(100, 10)

	view := m.View()
	if !strings.Contains(view, "No headings") {
		t.Errorf("expected empty-state message, got: %q", view)
	}
}

func TestParseHeadingsSkipsMathBlocks(t *testing.T) {
	// # Real
	//
	// $$
	// \begin{bmatrxi} a \\
	// b \end{bmatrix}
	// =
	// \begin{bmatrix} c \\
	// d \end{bmatrix}
	// $$
	//
	// ## After
	//
	source := "# Real\n\n$$\n\\begin{bmatrix} a \\\\ b \\end{bmatrix}\n=\n\\begin{bmatrix} c \\\\ d \\end{bmatrix}\n$$\n\n## After\n"
	want := []heading{
		{level: 1, text: "Real", sourceLine: 0, renderedLine: -1},
		{level: 2, text: "After", sourceLine: 8, renderedLine: -1},
	}

	got := parseHeadings([]byte(source))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseHeadings() = %#v, want %#v", got, want)
	}
}

func TestParseHeadingsDoesNotMaskMathInCodeFence(t *testing.T) {
	// # Real
	//
	// ```md
	// $$
	// =
	// $$
	// ```
	//
	// ## After
	//
	source := "# Real\n\n```md\n$$\n=\n$$\n```\n\n## After\n"
	want := []heading{
		{level: 1, text: "Real", sourceLine: 0, renderedLine: -1},
		{level: 2, text: "After", sourceLine: 8, renderedLine: -1},
	}

	got := parseHeadings([]byte(source))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseHeadings() = %#v, want %#v", got, want)
	}
}

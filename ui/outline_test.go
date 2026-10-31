package ui

import (
	"reflect"
	"testing"
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

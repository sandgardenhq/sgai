package main

import (
	"strings"
	"testing"
)

func TestExtractGoalDescription(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simpleHeading",
			in:   "# My Project Goal",
			want: "My Project Goal",
		},
		{
			name: "withFrontmatter",
			in:   "---\nflow: |\n  a -> b\n---\n# Improve UX for SGAI",
			want: "Improve UX for SGAI",
		},
		{
			name: "plainText",
			in:   "This is a simple description",
			want: "This is a simple description",
		},
		{
			name: "emptyContent",
			in:   "",
			want: "",
		},
		{
			name: "onlyFrontmatter",
			in:   "---\nkey: value\n---\n",
			want: "",
		},
		{
			name: "boldText",
			in:   "**Bold description** of the project",
			want: "Bold description of the project",
		},
		{
			name: "italicText",
			in:   "*Italic description* of the project",
			want: "Italic description of the project",
		},
		{
			name: "inlineCode",
			in:   "Use `go build` to compile",
			want: "Use go build to compile",
		},
		{
			name: "markdownLink",
			in:   "See [this project](https://example.com) for details",
			want: "See this project for details",
		},
		{
			name: "checkboxItem",
			in:   "- [ ] Implement feature X",
			want: "Implement feature X",
		},
		{
			name: "checkedItem",
			in:   "- [x] Implement feature X",
			want: "Implement feature X",
		},
		{
			name: "unorderedList",
			in:   "- First item in the list",
			want: "First item in the list",
		},
		{
			name: "orderedList",
			in:   "1. First item in the list",
			want: "First item in the list",
		},
		{
			name: "multipleHeadingLevels",
			in:   "## Second Level Heading",
			want: "Second Level Heading",
		},
		{
			name: "emptyLinesBeforeContent",
			in:   "\n\n\n# First Real Line",
			want: "First Real Line",
		},
		{
			name: "truncationAt256Chars",
			in:   strings.Repeat("A", 256),
			want: strings.Repeat("A", 255) + "...",
		},
		{
			name: "exactlyMaxLength",
			in:   strings.Repeat("B", 255),
			want: strings.Repeat("B", 255),
		},
		{
			name: "longWithFrontmatter",
			in:   "---\nkey: val\n---\n" + strings.Repeat("C", 300),
			want: strings.Repeat("C", 255) + "...",
		},
		{
			name: "mixedFormatting",
			in:   "# **Bold** and *italic* with `code`",
			want: "Bold and italic with code",
		},
		{
			name: "headingWithLink",
			in:   "# See [example](https://example.com) for details",
			want: "See example for details",
		},
		{
			name: "frontmatterThenEmptyThenContent",
			in:   "---\nflow: |\n  x -> y\n---\n\n# Real Content Here",
			want: "Real Content Here",
		},
		{
			name: "underscoreEmphasis",
			in:   "__underline emphasis__ text",
			want: "underline emphasis text",
		},
		{
			name: "starUnorderedList",
			in:   "* Star list item",
			want: "Star list item",
		},
		{
			name: "headingWithHashtagContent",
			in:   "# #hashtag in heading",
			want: "#hashtag in heading",
		},
		{
			name: "headingWithNestedHashes",
			in:   "# ## nested hashes",
			want: "## nested hashes",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractGoalDescription(tc.in)
			if got != tc.want {
				t.Errorf("extractGoalDescription() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestStripMarkdownFormatting(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "headingMarkers",
			in:   "### Heading Three",
			want: "Heading Three",
		},
		{
			name: "boldAndItalic",
			in:   "***bold italic***",
			want: "bold italic",
		},
		{
			name: "nestedEmphasis",
			in:   "**bold _and italic_**",
			want: "bold and italic",
		},
		{
			name: "plainText",
			in:   "just plain text",
			want: "just plain text",
		},
		{
			name: "headingWithHashtagContent",
			in:   "# #hashtag",
			want: "#hashtag",
		},
		{
			name: "headingWithNestedHashes",
			in:   "# ## nested",
			want: "## nested",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripMarkdownFormatting(tc.in)
			if got != tc.want {
				t.Errorf("stripMarkdownFormatting(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStripMarkdownLinks(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simpleLink",
			in:   "[text](https://example.com)",
			want: "text",
		},
		{
			name: "linkInContext",
			in:   "see [this](http://x.com) and [that](http://y.com)",
			want: "see this and that",
		},
		{
			name: "noLinks",
			in:   "no links here",
			want: "no links here",
		},
		{
			name: "bracketWithoutParen",
			in:   "[just brackets] here",
			want: "just brackets here",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripMarkdownLinks(tc.in)
			if got != tc.want {
				t.Errorf("stripMarkdownLinks(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

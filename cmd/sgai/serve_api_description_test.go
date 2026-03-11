package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripMarkdownFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plainText",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "heading",
			input:    "## Heading",
			expected: "Heading",
		},
		{
			name:     "checkbox",
			input:    "- [x] Task done",
			expected: "Task done",
		},
		{
			name:     "checkboxUnchecked",
			input:    "- [ ] Task pending",
			expected: "Task pending",
		},
		{
			name:     "listItem",
			input:    "- List item",
			expected: "List item",
		},
		{
			name:     "numberedList",
			input:    "1. First item",
			expected: "First item",
		},
		{
			name:     "link",
			input:    "[Click here](https://example.com)",
			expected: "Click here",
		},
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "bold text",
		},
		{
			name:     "italic",
			input:    "*italic text*",
			expected: "italic text",
		},
		{
			name:     "inlineCode",
			input:    "`code snippet`",
			expected: "code snippet",
		},
		{
			name:     "combined",
			input:    "## **Bold** and `code`",
			expected: "Bold and code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownFormatting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownHeadingPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "noHeading",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "h1",
			input:    "# Heading 1",
			expected: "Heading 1",
		},
		{
			name:     "h2",
			input:    "## Heading 2",
			expected: "Heading 2",
		},
		{
			name:     "h3",
			input:    "### Heading 3",
			expected: "Heading 3",
		},
		{
			name:     "multipleHashes",
			input:    "#### Heading 4",
			expected: "Heading 4",
		},
		{
			name:     "noSpaceAfterHash",
			input:    "###NoSpace",
			expected: "NoSpace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownHeadingPrefix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownCheckboxMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "checkedLowercase",
			input:    "- [x] Task completed",
			expected: "Task completed",
		},
		{
			name:     "checkedUppercase",
			input:    "- [X] Task completed",
			expected: "Task completed",
		},
		{
			name:     "unchecked",
			input:    "- [ ] Task pending",
			expected: "Task pending",
		},
		{
			name:     "noCheckbox",
			input:    "Regular text",
			expected: "Regular text",
		},
		{
			name:     "partialMatch",
			input:    "- [y] Not a checkbox",
			expected: "- [y] Not a checkbox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCheckboxMarkers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownListMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dashList",
			input:    "- List item",
			expected: "List item",
		},
		{
			name:     "asteriskList",
			input:    "* List item",
			expected: "List item",
		},
		{
			name:     "numberedList",
			input:    "1. First item",
			expected: "First item",
		},
		{
			name:     "multiDigitNumber",
			input:    "10. Tenth item",
			expected: "Tenth item",
		},
		{
			name:     "noMarker",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "justNumber",
			input:    "123",
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownListMarkers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownLinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simpleLink",
			input:    "[Click here](https://example.com)",
			expected: "Click here",
		},
		{
			name:     "linkWithText",
			input:    "Visit [our site](https://example.com) for more",
			expected: "Visit our site for more",
		},
		{
			name:     "multipleLinks",
			input:    "[Link 1](url1) and [Link 2](url2)",
			expected: "Link 1 and Link 2",
		},
		{
			name:     "noLink",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "unclosedBracket",
			input:    "[unclosed text",
			expected: "[unclosed text",
		},
		{
			name:     "bracketNoParen",
			input:    "[text] no url",
			expected: "text no url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownLinks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownEmphasis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "bold text",
		},
		{
			name:     "italic",
			input:    "*italic text*",
			expected: "italic text",
		},
		{
			name:     "boldItalic",
			input:    "***bold and italic***",
			expected: "bold and italic",
		},
		{
			name:     "underscoreBold",
			input:    "__bold text__",
			expected: "bold text",
		},
		{
			name:     "underscoreItalic",
			input:    "_italic text_",
			expected: "italic text",
		},
		{
			name:     "noEmphasis",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownEmphasis(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMarkdownInlineCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "inlineCode",
			input:    "`code snippet`",
			expected: "code snippet",
		},
		{
			name:     "multipleCode",
			input:    "use `foo` and `bar`",
			expected: "use foo and bar",
		},
		{
			name:     "noCode",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownInlineCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractGoalDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simpleGoal",
			input: `---
flow: |
  "agent1" -> "agent2"
---
# My Goal

This is a description.`,
			expected: "My Goal",
		},
		{
			name: "goalWithCheckbox",
			input: `---
flow: |
  "agent1" -> "agent2"
---
- [x] Completed task

This is a description.`,
			expected: "Completed task",
		},
		{
			name: "longDescription",
			input: `---
flow: |
  "agent1" -> "agent2"
---
` + string(make([]byte, 300)),
			expected: string(make([]byte, 255)) + "...",
		},
		{
			name: "emptyGoal",
			input: `---
flow: |
  "agent1" -> "agent2"
---

`,
			expected: "",
		},
		{
			name:     "noFrontmatter",
			input:    "# Heading\n\nDescription",
			expected: "Heading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGoalDescription(tt.input)
			if len(tt.expected) >= 256 {
				assert.Equal(t, tt.expected[:255]+"...", result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractGoalDescriptionComplex(t *testing.T) {
	content := "---\nflow: coordinator -> dev\nmodels:\n  coordinator: claude-opus-4\n---\n\n# Complex Goal Description\n\nThis is the body"
	result := extractGoalDescription(content)
	assert.Equal(t, "Complex Goal Description", result)
}

func TestExtractGoalDescriptionFromContent(t *testing.T) {
	content := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# My Project Description\nSome body"
	result := extractGoalDescription(content)
	assert.Equal(t, "My Project Description", result)
}

func TestExtractGoalDescriptionNoHeadingReturnsFirstLine(t *testing.T) {
	content := "---\nflow: |\n  \"a\" -> \"b\"\n---\nNo heading here"
	result := extractGoalDescription(content)
	assert.Equal(t, "No heading here", result)
}

func TestExtractGoalDescriptionSkipsEmptyHeading(t *testing.T) {
	content := "#\nActual title"
	result := extractGoalDescription(content)
	assert.Equal(t, "Actual title", result)
}

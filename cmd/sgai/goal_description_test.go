package main

import (
	"strings"
	"testing"
)

func TestGoalDescriptionFromContent(t *testing.T) {
	t.Run("simpleBody", func(t *testing.T) {
		content := "This is a simple goal. It has multiple sentences."
		got := goalDescriptionFromContent(content)
		if got != "This is a simple goal." {
			t.Errorf("got %q; want %q", got, "This is a simple goal.")
		}
	})

	t.Run("stripsFrontmatter", func(t *testing.T) {
		content := "---\nflow: |\n  agent1\n---\nBuild the feature. More details here."
		got := goalDescriptionFromContent(content)
		if got != "Build the feature." {
			t.Errorf("got %q; want %q", got, "Build the feature.")
		}
	})

	t.Run("stripsMarkdownHeading", func(t *testing.T) {
		content := "---\nkey: value\n---\n# My Project\n\nImplement user auth. This is important."
		got := goalDescriptionFromContent(content)
		if got != "My Project Implement user auth." {
			t.Errorf("got %q; want %q", got, "My Project Implement user auth.")
		}
	})

	t.Run("stripsBoldAndItalic", func(t *testing.T) {
		content := "Make **bold** and *italic* changes. Rest of text."
		got := goalDescriptionFromContent(content)
		if got != "Make bold and italic changes." {
			t.Errorf("got %q; want %q", got, "Make bold and italic changes.")
		}
	})

	t.Run("stripsLinks", func(t *testing.T) {
		content := "Check [this link](https://example.com) for details. More text."
		got := goalDescriptionFromContent(content)
		if got != "Check this link for details." {
			t.Errorf("got %q; want %q", got, "Check this link for details.")
		}
	})

	t.Run("stripsInlineCode", func(t *testing.T) {
		content := "Run the `make build` command. Then test."
		got := goalDescriptionFromContent(content)
		if got != "Run the make build command." {
			t.Errorf("got %q; want %q", got, "Run the make build command.")
		}
	})

	t.Run("truncatesLongContent", func(t *testing.T) {
		longText := strings.Repeat("a", 300) + ". Rest."
		got := goalDescriptionFromContent(longText)
		if len(got) != goalDescriptionMaxLength+3 {
			t.Errorf("length = %d; want %d (255 + '...')", len(got), goalDescriptionMaxLength+3)
		}
		if !strings.HasSuffix(got, "...") {
			t.Errorf("got %q; want suffix '...'", got)
		}
	})

	t.Run("atMaxLengthNoEllipsis", func(t *testing.T) {
		text := strings.Repeat("a", goalDescriptionMaxLength-1) + ". Rest."
		got := goalDescriptionFromContent(text)
		want := strings.Repeat("a", goalDescriptionMaxLength-1) + "."
		if got != want {
			t.Errorf("length = %d; want %d", len(got), len(want))
		}
		if strings.HasSuffix(got, "...") {
			t.Errorf("should not have ellipsis for content at max length")
		}
	})

	t.Run("emptyContent", func(t *testing.T) {
		got := goalDescriptionFromContent("")
		if got != "" {
			t.Errorf("got %q; want empty string", got)
		}
	})

	t.Run("frontmatterOnly", func(t *testing.T) {
		content := "---\nflow: |\n  agent1\n---\n"
		got := goalDescriptionFromContent(content)
		if got != "" {
			t.Errorf("got %q; want empty string", got)
		}
	})

	t.Run("noPeriodInText", func(t *testing.T) {
		content := "This is the entire goal with no period"
		got := goalDescriptionFromContent(content)
		if got != "This is the entire goal with no period" {
			t.Errorf("got %q; want %q", got, "This is the entire goal with no period")
		}
	})

	t.Run("periodAtEndOfFile", func(t *testing.T) {
		content := "Build a REST API."
		got := goalDescriptionFromContent(content)
		if got != "Build a REST API." {
			t.Errorf("got %q; want %q", got, "Build a REST API.")
		}
	})

	t.Run("periodFollowedByNewline", func(t *testing.T) {
		content := "Build the app.\nMore details follow."
		got := goalDescriptionFromContent(content)
		if got != "Build the app." {
			t.Errorf("got %q; want %q", got, "Build the app.")
		}
	})

	t.Run("stripsCheckboxes", func(t *testing.T) {
		content := "- [ ] Add tests. More items follow."
		got := goalDescriptionFromContent(content)
		if got != "Add tests." {
			t.Errorf("got %q; want %q", got, "Add tests.")
		}
	})

	t.Run("realWorldGoalMd", func(t *testing.T) {
		content := `---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
---

We are going to make a set of changes to experiment with what.

- [ ] On the left bar, we are going to have only forked workspaces
  - [ ] in the Root Repository in Forked Mode, there will be a rich editor
`
		got := goalDescriptionFromContent(content)
		if got != "We are going to make a set of changes to experiment with what." {
			t.Errorf("got %q; want %q", got, "We are going to make a set of changes to experiment with what.")
		}
	})
}

func TestExtractFirstSentence(t *testing.T) {
	t.Run("periodFollowedBySpace", func(t *testing.T) {
		got := extractFirstSentence("First sentence. Second sentence.")
		if got != "First sentence." {
			t.Errorf("got %q; want %q", got, "First sentence.")
		}
	})

	t.Run("periodFollowedByNewline", func(t *testing.T) {
		got := extractFirstSentence("First sentence.\nSecond sentence.")
		if got != "First sentence." {
			t.Errorf("got %q; want %q", got, "First sentence.")
		}
	})

	t.Run("periodAtEnd", func(t *testing.T) {
		got := extractFirstSentence("Only sentence.")
		if got != "Only sentence." {
			t.Errorf("got %q; want %q", got, "Only sentence.")
		}
	})

	t.Run("noPeriod", func(t *testing.T) {
		got := extractFirstSentence("No period here")
		if got != "No period here" {
			t.Errorf("got %q; want %q", got, "No period here")
		}
	})

	t.Run("emptyString", func(t *testing.T) {
		got := extractFirstSentence("")
		if got != "" {
			t.Errorf("got %q; want empty string", got)
		}
	})

	t.Run("periodNotFollowedBySpaceOrNewline", func(t *testing.T) {
		got := extractFirstSentence("file.txt is here. Next sentence.")
		if got != "file.txt is here." {
			t.Errorf("got %q; want %q", got, "file.txt is here.")
		}
	})
}

func TestStripMarkdownFormatting(t *testing.T) {
	t.Run("bold", func(t *testing.T) {
		got := stripMarkdownFormatting("this is **bold** text")
		if got != "this is bold text" {
			t.Errorf("got %q; want %q", got, "this is bold text")
		}
	})

	t.Run("italic", func(t *testing.T) {
		got := stripMarkdownFormatting("this is *italic* text")
		if got != "this is italic text" {
			t.Errorf("got %q; want %q", got, "this is italic text")
		}
	})

	t.Run("link", func(t *testing.T) {
		got := stripMarkdownFormatting("[click here](https://example.com)")
		if got != "click here" {
			t.Errorf("got %q; want %q", got, "click here")
		}
	})

	t.Run("inlineCode", func(t *testing.T) {
		got := stripMarkdownFormatting("use `fmt.Println` here")
		if got != "use fmt.Println here" {
			t.Errorf("got %q; want %q", got, "use fmt.Println here")
		}
	})

	t.Run("image", func(t *testing.T) {
		got := stripMarkdownFormatting("see ![alt text](image.png) here")
		if got != "see alt text here" {
			t.Errorf("got %q; want %q", got, "see alt text here")
		}
	})

	t.Run("heading", func(t *testing.T) {
		got := stripMarkdownFormatting("## My Heading")
		if got != "My Heading" {
			t.Errorf("got %q; want %q", got, "My Heading")
		}
	})

	t.Run("plainText", func(t *testing.T) {
		got := stripMarkdownFormatting("just plain text")
		if got != "just plain text" {
			t.Errorf("got %q; want %q", got, "just plain text")
		}
	})
}

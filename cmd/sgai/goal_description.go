package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const goalDescriptionMaxLength = 255

func goalDescriptionFromContent(content string) string {
	body := stripFrontmatter(content)
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	firstSentence := extractFirstSentence(body)
	plaintext := stripMarkdownFormatting(firstSentence)
	plaintext = strings.TrimSpace(plaintext)
	if len(plaintext) > goalDescriptionMaxLength {
		return plaintext[:goalDescriptionMaxLength] + "..."
	}
	return plaintext
}

func extractFirstSentence(text string) string {
	for i := 0; i < len(text)-1; i++ {
		if text[i] == '.' && (text[i+1] == ' ' || text[i+1] == '\n' || text[i+1] == '\r') {
			return text[:i+1]
		}
	}
	if len(text) > 0 && text[len(text)-1] == '.' {
		return text
	}
	return text
}

var (
	reMarkdownHeadings    = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reMarkdownBold3       = regexp.MustCompile(`\*{3}(.+?)\*{3}`)
	reMarkdownBold2       = regexp.MustCompile(`\*{2}(.+?)\*{2}`)
	reMarkdownItalic      = regexp.MustCompile(`\*(.+?)\*`)
	reMarkdownUndBold3    = regexp.MustCompile(`_{3}(.+?)_{3}`)
	reMarkdownUndBold2    = regexp.MustCompile(`_{2}(.+?)_{2}`)
	reMarkdownUndItalic   = regexp.MustCompile(`_(.+?)_`)
	reMarkdownLinks       = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	reMarkdownInlineCode  = regexp.MustCompile("`([^`]+)`")
	reMarkdownImages      = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	reMarkdownCheckboxes  = regexp.MustCompile(`(?m)^\s*-\s*\[[ xX]\]\s*`)
	reMarkdownListMarkers = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	reMultipleSpaces      = regexp.MustCompile(`\s+`)
)

func stripMarkdownFormatting(text string) string {
	result := text
	result = reMarkdownImages.ReplaceAllString(result, "$1")
	result = reMarkdownLinks.ReplaceAllString(result, "$1")
	result = reMarkdownInlineCode.ReplaceAllString(result, "$1")
	result = reMarkdownBold3.ReplaceAllString(result, "$1")
	result = reMarkdownBold2.ReplaceAllString(result, "$1")
	result = reMarkdownItalic.ReplaceAllString(result, "$1")
	result = reMarkdownUndBold3.ReplaceAllString(result, "$1")
	result = reMarkdownUndBold2.ReplaceAllString(result, "$1")
	result = reMarkdownUndItalic.ReplaceAllString(result, "$1")
	result = reMarkdownHeadings.ReplaceAllString(result, "")
	result = reMarkdownCheckboxes.ReplaceAllString(result, "")
	result = reMarkdownListMarkers.ReplaceAllString(result, "")
	result = reMultipleSpaces.ReplaceAllString(result, " ")
	return result
}

func readGoalDescription(dir string) string {
	data, errRead := os.ReadFile(filepath.Join(dir, "GOAL.md"))
	if errRead != nil {
		return ""
	}
	return goalDescriptionFromContent(string(data))
}

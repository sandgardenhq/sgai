package main

import "strings"

func extractGoalDescription(fullContent string) string {
	body := stripFrontmatter(fullContent)
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		plain := stripMarkdownFormatting(trimmed)
		plain = strings.TrimSpace(plain)
		if plain == "" {
			continue
		}
		if len(plain) >= 256 {
			return plain[:255] + "..."
		}
		return plain
	}
	return ""
}

func stripMarkdownFormatting(s string) string {
	s = stripMarkdownHeadingPrefix(s)
	s = stripMarkdownCheckboxMarkers(s)
	s = stripMarkdownListMarkers(s)
	s = stripMarkdownLinks(s)
	s = stripMarkdownEmphasis(s)
	s = stripMarkdownInlineCode(s)
	return s
}

func stripMarkdownHeadingPrefix(s string) string {
	for len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	return strings.TrimLeft(s, " ")
}

func stripMarkdownCheckboxMarkers(s string) string {
	for _, prefix := range []string{"- [x] ", "- [X] ", "- [ ] "} {
		if strings.HasPrefix(s, prefix) {
			return s[len(prefix):]
		}
	}
	return s
}

func stripMarkdownListMarkers(s string) string {
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		return s[2:]
	}
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 && i+1 < len(s) && s[i+1] == ' ' {
			return s[i+2:]
		}
		break
	}
	return s
}

func stripMarkdownLinks(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '[' {
			closeBracket := strings.Index(s[i:], "]")
			if closeBracket == -1 {
				result.WriteByte(s[i])
				i++
				continue
			}
			absClose := i + closeBracket
			linkText := s[i+1 : absClose]
			if absClose+1 < len(s) && s[absClose+1] == '(' {
				closeParen := strings.Index(s[absClose+1:], ")")
				if closeParen != -1 {
					result.WriteString(linkText)
					i = absClose + 1 + closeParen + 1
					continue
				}
			}
			result.WriteString(linkText)
			i = absClose + 1
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func stripMarkdownEmphasis(s string) string {
	s = strings.ReplaceAll(s, "***", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "___", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func stripMarkdownInlineCode(s string) string {
	return strings.ReplaceAll(s, "`", "")
}

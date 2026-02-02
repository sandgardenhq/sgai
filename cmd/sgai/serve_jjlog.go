package main

import (
	"fmt"
	"html/template"
	"net/url"
	"os/exec"
	"slices"
	"strings"
)

type jjCommit struct {
	ChangeID      string
	CommitID      string
	Workspaces    []string
	Timestamp     string
	Bookmarks     []string
	Description   string
	GraphChar     string
	HasLine       bool
	GraphLines    []string
	TrailingGraph []string
}

const jjLogTemplate = `change_id.short(8) ++ " " ++ commit_id.short(8) ++ " " ++ if(working_copies, working_copies.map(|wc| wc.name()).join(" ") ++ " ", "") ++ author.timestamp().ago() ++ if(bookmarks, " " ++ bookmarks.join(" "), "") ++ "\n  " ++ coalesce(description.first_line(), "(no description)") ++ "\n"`

var timestampUnits = []string{"second", "seconds", "minute", "minutes", "hour", "hours", "day", "days", "week", "weeks", "month", "months", "year", "years", "ago"}

func runJJLogForRoot(dir string) []jjCommit {
	revset := `::@ | working_copies()`
	cmd := exec.Command("jj", "log", "-r", revset, "-T", jjLogTemplate)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func runJJLogForFork(dir string) []jjCommit {
	revset := `::@`
	cmd := exec.Command("jj", "log", "-r", revset, "-T", jjLogTemplate)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func parseJJLogOutput(output string) []jjCommit {
	var commits []jjCommit
	lines := linesWithTrailingEmpty(output)

	var currentCommit *jjCommit
	for i, line := range lines {
		if line == "" {
			continue
		}

		if isCommitHeaderLine(line) {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}
			currentCommit = parseCommitHeader(line)
			currentCommit.HasLine = hasNextCommit(lines, i)
			currentCommit.GraphLines = []string{extractGraphPrefix(line)}
		} else if currentCommit != nil {
			strippedContent := stripGraphPrefix(line)
			graphPrefix := extractGraphPrefix(line)

			if currentCommit.Description == "" && strippedContent != "" {
				currentCommit.Description = strings.TrimSpace(strippedContent)
			}

			if graphPrefix != "" {
				currentCommit.TrailingGraph = append(currentCommit.TrailingGraph, graphPrefix)
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits
}

func isCommitMarker(r rune) bool {
	return r == '○' || r == '×' || r == '@' || r == '◆' || r == '~'
}

func isCommitHeaderLine(line string) bool {
	if len(line) < 3 {
		return false
	}
	for _, r := range line {
		if isCommitMarker(r) {
			return true
		}
		if !isGraphChar(r) {
			return false
		}
	}
	return false
}

func isGraphChar(r rune) bool {
	return r == '│' || r == '├' || r == '─' || r == '┘' || r == ' '
}

func stripGraphPrefix(line string) string {
	runes := []rune(line)
	for i, r := range runes {
		if !isGraphChar(r) {
			return string(runes[i:])
		}
	}
	return ""
}

func extractGraphPrefix(line string) string {
	runes := []rune(line)
	for i, r := range runes {
		if !isGraphChar(r) && !isCommitMarker(r) {
			return strings.TrimRight(string(runes[:i]), " ")
		}
	}
	return strings.TrimRight(line, " ")
}

func hasNextCommit(lines []string, currentIdx int) bool {
	return slices.ContainsFunc(lines[currentIdx+1:], isCommitHeaderLine)
}

func findCommitMarker(line string) (marker rune, restOfLine string) {
	runes := []rune(line)
	for i, r := range runes {
		if isCommitMarker(r) {
			return r, strings.TrimSpace(string(runes[i+1:]))
		}
	}
	return 0, line
}

func parseCommitHeader(line string) *jjCommit {
	commit := &jjCommit{}

	marker, rest := findCommitMarker(line)
	if marker == 0 {
		return commit
	}
	commit.GraphChar = string(marker)

	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return commit
	}

	commit.ChangeID = parts[0]
	commit.CommitID = parts[1]

	remaining := parts[2:]

	for i := 0; i < len(remaining); i++ {
		part := remaining[i]

		if isTimestamp(part) {
			commit.Timestamp = part
			for i+1 < len(remaining) && isTimestampUnit(remaining[i+1]) {
				commit.Timestamp += " " + remaining[i+1]
				i++
			}
			continue
		}

		if strings.HasSuffix(part, "*") || isBookmark(part) {
			commit.Bookmarks = append(commit.Bookmarks, part)
			continue
		}

		if !isTimestamp(part) && !isTimestampUnit(part) && len(commit.Workspaces) == 0 {
			commit.Workspaces = append(commit.Workspaces, part)
		}
	}

	return commit
}

func isTimestamp(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	return first >= '0' && first <= '9'
}

func isTimestampUnit(s string) bool {
	for _, u := range timestampUnits {
		if strings.HasPrefix(s, u) {
			return true
		}
	}
	return false
}

func isBookmark(s string) bool {
	return strings.Contains(s, "@") || strings.Contains(s, "/")
}

func renderJJLogHTML(commits []jjCommit, currentWorkspace string) string {
	if len(commits) == 0 {
		return `<article class="jj-empty"><p>No commits found</p></article>`
	}

	var buf strings.Builder
	buf.WriteString(`<article class="jj-log-article"><div class="jj-log">`)

	for i, commit := range commits {
		isLast := i == len(commits)-1
		buf.WriteString(renderCommitHTML(commit, isLast, currentWorkspace))
	}

	buf.WriteString(`</div></article>`)
	return buf.String()
}

func renderCommitHTML(commit jjCommit, _ bool, currentWorkspace string) string {
	var buf strings.Builder

	commitClass := "jj-commit"
	if commit.GraphChar == "@" || slices.Contains(commit.Workspaces, currentWorkspace) {
		commitClass += " current"
	}
	buf.WriteString(fmt.Sprintf(`<div class="%s">`, commitClass))

	buf.WriteString(`<div class="jj-graph-tree">`)
	buf.WriteString(`<pre class="jj-graph-pre">`)
	var graphContent strings.Builder
	if len(commit.GraphLines) > 0 {
		graphContent.WriteString(commit.GraphLines[0])
	}
	if len(commit.TrailingGraph) > 0 {
		for _, tg := range commit.TrailingGraph {
			graphContent.WriteString("\n")
			graphContent.WriteString(tg)
		}
	}
	buf.WriteString(template.HTMLEscapeString(graphContent.String()))
	buf.WriteString(`</pre>`)
	buf.WriteString(`</div>`)

	buf.WriteString(`<div class="jj-content">`)

	buf.WriteString(`<div class="jj-meta">`)
	buf.WriteString(`<code class="jj-change-id">`)
	buf.WriteString(template.HTMLEscapeString(commit.ChangeID))
	buf.WriteString(`</code>`)
	buf.WriteString(`<code class="jj-commit-id">`)
	buf.WriteString(template.HTMLEscapeString(commit.CommitID))
	buf.WriteString(`</code>`)

	if len(commit.Workspaces) > 0 {
		for _, ws := range commit.Workspaces {
			wsClass := "jj-workspace-badge"
			if ws == currentWorkspace {
				wsClass += " current"
				buf.WriteString(fmt.Sprintf(`<mark class="%s">%s</mark>`, wsClass, template.HTMLEscapeString(ws)))
			} else {
				buf.WriteString(fmt.Sprintf(`<a href="/trees?workspace=%s&tab=commits" class="%s" title="Navigate to %s fork">%s</a>`,
					url.QueryEscape(ws),
					wsClass,
					template.HTMLEscapeString(ws),
					template.HTMLEscapeString(ws)))
			}
		}
	}

	if len(commit.Bookmarks) > 0 {
		for _, bm := range commit.Bookmarks {
			buf.WriteString(fmt.Sprintf(`<kbd class="jj-bookmark-badge">%s</kbd>`, template.HTMLEscapeString(bm)))
		}
	}

	if commit.Timestamp != "" {
		buf.WriteString(fmt.Sprintf(`<small class="jj-timestamp">%s</small>`, template.HTMLEscapeString(commit.Timestamp)))
	}
	buf.WriteString(`</div>`)

	description := commit.Description
	if description == "" || description == "(no description)" {
		buf.WriteString(`<p class="jj-description empty">(no description)</p>`)
	} else {
		buf.WriteString(fmt.Sprintf(`<p class="jj-description">%s</p>`, template.HTMLEscapeString(description)))
	}

	buf.WriteString(`</div>`)
	buf.WriteString(`</div>`)

	return buf.String()
}

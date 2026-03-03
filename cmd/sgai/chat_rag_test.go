package main

import (
	"slices"
	"testing"
)

func TestExtractKeywords(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "simple text",
			input:    "How do I create a new workspace?",
			contains: []string{"create", "workspace"},
		},
		{
			name:     "filters stop words",
			input:    "What is the workflow and how does it work?",
			contains: []string{"workflow"},
		},
		{
			name:     "extracts technical terms",
			input:    "GOAL.md agent coordinator SGAI",
			contains: []string{"goal", "agent", "coordinator", "sgai"},
		},
		{
			name:  "empty input",
			input: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractKeywords(tc.input)
			if tc.input == "" && len(got) != 0 {
				t.Errorf("extractKeywords(%q) returned %d keywords; want 0", tc.input, len(got))
			}
			for _, want := range tc.contains {
				if !slices.Contains(got, want) {
					t.Errorf("extractKeywords(%q) missing %q; got %v", tc.input, want, got)
				}
			}
		})
	}
}

func TestIsStopWord(t *testing.T) {
	stopWordTests := []string{"the", "and", "for", "with", "from"}
	for _, word := range stopWordTests {
		if !isStopWord(word) {
			t.Errorf("isStopWord(%q) = false; want true", word)
		}
	}

	nonStopWordTests := []string{"agent", "workflow", "sgai", "coordinator"}
	for _, word := range nonStopWordTests {
		if isStopWord(word) {
			t.Errorf("isStopWord(%q) = true; want false", word)
		}
	}
}

func TestSplitByHeadings(t *testing.T) {
	content := `# Main Title

Some intro text.

## Section One

Content of section one.

## Section Two

Content of section two.
`

	sections := splitByHeadings(content)
	if len(sections) != 3 {
		t.Errorf("splitByHeadings returned %d sections; want 3", len(sections))
	}
}

func TestFormatRetrievedDocs(t *testing.T) {
	chunks := []docChunk{
		{
			Content:  "This is chunk one.",
			Source:   "docs/reference/cli.md",
			Keywords: []string{"cli", "command"},
		},
		{
			Content:  "This is chunk two.",
			Source:   "docs/AGENTS.md",
			Keywords: []string{"agent", "workflow"},
		},
	}

	result := formatRetrievedDocs(chunks)
	if result == "" {
		t.Error("formatRetrievedDocs returned empty string")
	}

	if !contains(result, "docs/reference/cli.md") {
		t.Error("formatRetrievedDocs missing source reference")
	}

	if !contains(result, "This is chunk one.") {
		t.Error("formatRetrievedDocs missing content")
	}

	if !contains(result, "---") {
		t.Error("formatRetrievedDocs missing separator")
	}
}

func TestFormatRetrievedDocsEmpty(t *testing.T) {
	result := formatRetrievedDocs(nil)
	if result != "" {
		t.Errorf("formatRetrievedDocs(nil) = %q; want empty string", result)
	}

	result = formatRetrievedDocs([]docChunk{})
	if result != "" {
		t.Errorf("formatRetrievedDocs([]) = %q; want empty string", result)
	}
}

func TestComputeTFIDFScore(t *testing.T) {
	allChunks := []docChunk{
		{Keywords: []string{"agent", "workflow", "sgai"}},
		{Keywords: []string{"agent", "coordinator"}},
		{Keywords: []string{"workflow", "goal"}},
	}

	queryKeywords := []string{"agent"}
	chunkKeywords := []string{"agent", "workflow", "sgai"}

	score := computeTFIDFScore(queryKeywords, chunkKeywords, allChunks)
	if score <= 0 {
		t.Error("computeTFIDFScore returned non-positive score for matching keyword")
	}

	noMatchScore := computeTFIDFScore([]string{"nonexistent"}, chunkKeywords, allChunks)
	if noMatchScore != 0 {
		t.Errorf("computeTFIDFScore returned %f for non-matching keyword; want 0", noMatchScore)
	}
}

func TestCountDocumentFrequency(t *testing.T) {
	chunks := []docChunk{
		{Keywords: []string{"agent", "workflow"}},
		{Keywords: []string{"agent", "coordinator"}},
		{Keywords: []string{"workflow", "goal"}},
	}

	agentFreq := countDocumentFrequency("agent", chunks)
	if agentFreq != 2 {
		t.Errorf("countDocumentFrequency(\"agent\") = %d; want 2", agentFreq)
	}

	unknownFreq := countDocumentFrequency("unknown", chunks)
	if unknownFreq != 0 {
		t.Errorf("countDocumentFrequency(\"unknown\") = %d; want 0", unknownFreq)
	}
}

func TestLoadChatDocumentation(t *testing.T) {
	chunks := loadChatDocumentation()
	if len(chunks) == 0 {
		t.Skip("no documentation chunks loaded (expected in test environment without embedded docs)")
	}

	for i, chunk := range chunks {
		if chunk.Source == "" {
			t.Errorf("chunk[%d] has empty source", i)
		}
		if chunk.Content == "" {
			t.Errorf("chunk[%d] has empty content", i)
		}
	}
}

func TestRetrieveRelevantChunks(t *testing.T) {
	chunks := retrieveRelevantChunks("agent workflow", 5)
	if chunks == nil {
		t.Log("retrieveRelevantChunks returned nil (may be expected without embedded docs)")
		return
	}

	if len(chunks) > 5 {
		t.Errorf("retrieveRelevantChunks returned %d chunks; want <= 5", len(chunks))
	}
}

func TestRetrieveRelevantChunksEmptyQuery(t *testing.T) {
	chunks := retrieveRelevantChunks("", 5)
	if chunks != nil {
		t.Errorf("retrieveRelevantChunks(\"\", 5) = %v; want nil", chunks)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

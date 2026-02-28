package main

import (
	"io/fs"
	"math"
	"slices"
	"strings"
	"sync"
)

// docChunk represents a searchable chunk of documentation.
type docChunk struct {
	Content  string
	Source   string
	Keywords []string
}

var (
	chatDocsOnce   sync.Once
	chatDocsChunks []docChunk
)

func loadChatDocumentation() []docChunk {
	chatDocsOnce.Do(func() {
		chatDocsChunks = parseChatDocs()
	})
	return chatDocsChunks
}

func parseChatDocs() []docChunk {
	var chunks []docChunk

	errWalk := fs.WalkDir(chatDocsFS, "docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, errRead := fs.ReadFile(chatDocsFS, path)
		if errRead != nil {
			return nil
		}

		stripped := stripFrontmatter(string(content))
		docChunks := splitIntoChunks(stripped, path)
		chunks = append(chunks, docChunks...)
		return nil
	})
	if errWalk != nil {
		return nil
	}

	return chunks
}

func splitIntoChunks(content, source string) []docChunk {
	sections := splitByHeadings(content)
	chunks := make([]docChunk, 0, len(sections))

	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		keywords := extractKeywords(section)
		chunks = append(chunks, docChunk{
			Content:  section,
			Source:   source,
			Keywords: keywords,
		})
	}

	return chunks
}

func splitByHeadings(content string) []string {
	var sections []string
	var current strings.Builder
	headingPrefix := "##"

	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(line, headingPrefix) && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		sections = append(sections, current.String())
	}

	return sections
}

func extractKeywords(text string) []string {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, isNonAlphaNum)

	keywordSet := make(map[string]struct{})
	for _, word := range words {
		if len(word) >= 3 && !isStopWord(word) {
			keywordSet[word] = struct{}{}
		}
	}

	keywords := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		keywords = append(keywords, kw)
	}
	slices.Sort(keywords)
	return keywords
}

func isNonAlphaNum(r rune) bool {
	isLower := r >= 'a' && r <= 'z'
	isUpper := r >= 'A' && r <= 'Z'
	isDigit := r >= '0' && r <= '9'
	return !isLower && !isUpper && !isDigit
}

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "but": true,
	"not": true, "you": true, "all": true, "can": true, "had": true,
	"her": true, "was": true, "one": true, "our": true, "out": true,
	"has": true, "his": true, "its": true, "this": true, "that": true,
	"with": true, "from": true, "have": true, "they": true, "will": true,
	"what": true, "when": true, "make": true, "like": true, "just": true,
	"into": true, "year": true, "your": true, "than": true, "them": true,
	"been": true, "would": true, "which": true, "their": true, "about": true,
}

func isStopWord(word string) bool {
	return stopWords[word]
}

// retrieveRelevantChunks returns the topK most relevant documentation chunks for a query.
func retrieveRelevantChunks(query string, topK int) []docChunk {
	chunks := loadChatDocumentation()
	if len(chunks) == 0 {
		return nil
	}

	queryKeywords := extractKeywords(query)
	if len(queryKeywords) == 0 {
		return nil
	}

	type scoredChunk struct {
		chunk docChunk
		score float64
	}

	scored := make([]scoredChunk, 0, len(chunks))
	for _, chunk := range chunks {
		score := computeTFIDFScore(queryKeywords, chunk.Keywords, chunks)
		if score > 0 {
			scored = append(scored, scoredChunk{chunk: chunk, score: score})
		}
	}

	slices.SortFunc(scored, func(a, b scoredChunk) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return 0
	})

	if len(scored) > topK {
		scored = scored[:topK]
	}

	result := make([]docChunk, len(scored))
	for i, sc := range scored {
		result[i] = sc.chunk
	}
	return result
}

func computeTFIDFScore(queryKeywords, chunkKeywords []string, allChunks []docChunk) float64 {
	chunkKeywordSet := make(map[string]bool)
	for _, kw := range chunkKeywords {
		chunkKeywordSet[kw] = true
	}

	var score float64
	for _, qkw := range queryKeywords {
		if !chunkKeywordSet[qkw] {
			continue
		}

		docFreq := countDocumentFrequency(qkw, allChunks)
		if docFreq == 0 {
			continue
		}

		idf := math.Log(float64(len(allChunks)+1) / float64(docFreq+1))
		score += idf
	}

	return score
}

func countDocumentFrequency(keyword string, chunks []docChunk) int {
	count := 0
	for _, chunk := range chunks {
		if slices.Contains(chunk.Keywords, keyword) {
			count++
		}
	}
	return count
}

// formatRetrievedDocs formats retrieved chunks for inclusion in a prompt.
func formatRetrievedDocs(chunks []docChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, chunk := range chunks {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString("Source: ")
		sb.WriteString(chunk.Source)
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(chunk.Content))
		sb.WriteString("\n")
	}
	return sb.String()
}

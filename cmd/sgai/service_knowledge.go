package main

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

type listAgentsResult struct {
	Agents []apiAgentEntry
}

func (s *Server) listAgentsService(workspacePath string) listAgentsResult {
	return listAgentsResult{Agents: collectAgents(workspacePath)}
}

type listSkillsResult struct {
	Categories []apiSkillCategory
}

func (s *Server) listSkillsService(workspacePath string) listSkillsResult {
	return listSkillsResult{Categories: collectSkillCategories(workspacePath)}
}

type skillDetailResult struct {
	Name       string
	FullPath   string
	Content    string
	RawContent string
	Found      bool
}

func (s *Server) skillDetailService(workspacePath, skillName string) skillDetailResult {
	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	skillFilePath := skillName + "/SKILL.md"
	content, errRead := fs.ReadFile(skillsFS, skillFilePath)
	if errRead != nil {
		return skillDetailResult{Found: false}
	}

	stripped := stripFrontmatter(string(content))
	rendered, errRender := renderMarkdown([]byte(stripped))
	if errRender != nil {
		rendered = stripped
	}

	return skillDetailResult{
		Name:       path.Base(skillName),
		FullPath:   skillName,
		Content:    rendered,
		RawContent: stripped,
		Found:      true,
	}
}

type listSnippetsResult struct {
	Languages []apiLanguageCategory
}

func (s *Server) listSnippetsService(workspacePath string) listSnippetsResult {
	languages := convertSnippetLanguages(gatherSnippetsByLanguage(workspacePath))
	return listSnippetsResult{Languages: languages}
}

type snippetsByLanguageResult struct {
	Language string
	Snippets []apiSnippetEntry
	Found    bool
}

func (s *Server) snippetsByLanguageService(workspacePath, lang string) snippetsByLanguageResult {
	allLanguages := convertSnippetLanguages(gatherSnippetsByLanguage(workspacePath))
	for _, lc := range allLanguages {
		if lc.Name == lang {
			return snippetsByLanguageResult{Language: lc.Name, Snippets: lc.Snippets, Found: true}
		}
	}
	return snippetsByLanguageResult{Found: false}
}

type snippetDetailResult struct {
	Name        string
	FileName    string
	Language    string
	Description string
	WhenToUse   string
	Content     string
	Found       bool
}

func (s *Server) snippetDetailService(workspacePath, lang, fileName string) snippetDetailResult {
	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	extensions := []string{".go", ".html", ".css", ".js", ".ts", ".py", ".sh", ".yaml", ".yml", ".json", ".md", ".sql", ".txt", ""}
	var content []byte
	for _, ext := range extensions {
		filePath := lang + "/" + fileName + ext
		var errRead error
		content, errRead = fs.ReadFile(snippetsFS, filePath)
		if errRead == nil {
			break
		}
	}

	if content == nil {
		return snippetDetailResult{Found: false}
	}

	fm := parseFrontmatterMap(content)
	name := fm["name"]
	if name == "" {
		name = fileName
	}

	return snippetDetailResult{
		Name:        name,
		FileName:    fileName,
		Language:    lang,
		Description: fm["description"],
		WhenToUse:   fm["when_to_use"],
		Content:     stripFrontmatter(string(content)),
		Found:       true,
	}
}

type listModelsResult struct {
	Models       []apiModelEntry
	DefaultModel string
}

func (s *Server) listModelsService(workspaceName string) (listModelsResult, error) {
	catalog, errModels := fetchValidModels()
	if errModels != nil {
		return listModelsResult{}, errModels
	}

	defaultModel := s.coordinatorModelFromWorkspace(workspaceName)
	return listModelsResult{Models: modelEntries(catalog), DefaultModel: defaultModel}, nil
}

package main

import (
	"html/template"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type skillData struct {
	Name        string
	FullPath    string
	Description string
}

type categoryData struct {
	Name   string
	Skills []skillData
}

func (s *Server) handleWorkspaceSkills(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	categories := make(map[string][]skillData)

	err := fs.WalkDir(skillsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		content, errRead := fs.ReadFile(skillsFS, path)
		if errRead != nil {
			return nil
		}
		skillPath := strings.TrimSuffix(path, "/SKILL.md")
		parts := strings.Split(skillPath, "/")
		var category string
		var name string
		if len(parts) > 1 {
			category = parts[0]
			name = strings.Join(parts[1:], "/")
		} else {
			category = ""
			name = skillPath
		}
		desc := extractFrontmatterDescription(string(content))
		categories[category] = append(categories[category], skillData{
			Name:        name,
			FullPath:    skillPath,
			Description: desc,
		})
		return nil
	})
	if err != nil {
		categories = nil
	}

	var categoryList []categoryData
	categoryNames := slices.Sorted(maps.Keys(categories))

	for _, categoryName := range categoryNames {
		skills := categories[categoryName]
		slices.SortFunc(skills, func(a, b skillData) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		displayName := categoryName
		if displayName == "" {
			displayName = "General"
		}
		categoryList = append(categoryList, categoryData{
			Name:   displayName,
			Skills: skills,
		})
	}

	dirName := filepath.Base(workspacePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("skills.html"), struct {
		Categories []categoryData
		DirName    string
	}{categoryList, dirName})
}

func (s *Server) handleWorkspaceSkillDetail(w http.ResponseWriter, r *http.Request, workspacePath, skillPath string) {
	if skillPath == "" {
		dirName := filepath.Base(workspacePath)
		http.Redirect(w, r, "/workspaces/"+dirName+"/skills", http.StatusSeeOther)
		return
	}

	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	skillFilePath := skillPath + "/SKILL.md"
	content, err := fs.ReadFile(skillsFS, skillFilePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := filepath.Base(skillPath)
	stripped := stripFrontmatter(string(content))
	rendered, errRender := renderMarkdown([]byte(stripped))
	if errRender != nil {
		rendered = stripped
	}

	dirName := filepath.Base(workspacePath)
	data := struct {
		Name       string
		FullPath   string
		Content    template.HTML
		RawContent string
		DirName    string
	}{
		Name:       name,
		FullPath:   skillPath,
		Content:    template.HTML(rendered),
		RawContent: stripped,
		DirName:    dirName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("skill_detail.html"), data)
}

type snippetData struct {
	Name        string
	FileName    string
	FullPath    string
	Description string
	Language    string
}

type languageCategory struct {
	Name     string
	Snippets []snippetData
}

func (s *Server) handleWorkspaceSnippets(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	languages := make(map[string][]snippetData)

	err := fs.WalkDir(snippetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, errRead := fs.ReadFile(snippetsFS, path)
		if errRead != nil {
			return nil
		}

		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return nil
		}

		language := parts[0]
		filename := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

		fm := parseFrontmatterMap(content)
		name := fm["name"]
		if name == "" {
			name = filename
		}
		description := fm["description"]

		languages[language] = append(languages[language], snippetData{
			Name:        name,
			FileName:    filename,
			FullPath:    language + "/" + filename,
			Description: description,
			Language:    language,
		})

		return nil
	})
	if err != nil {
		languages = nil
	}

	var languageList []languageCategory
	languageNames := slices.Sorted(maps.Keys(languages))

	for _, languageName := range languageNames {
		snippets := languages[languageName]
		slices.SortFunc(snippets, func(a, b snippetData) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		languageList = append(languageList, languageCategory{
			Name:     languageName,
			Snippets: snippets,
		})
	}

	dirName := filepath.Base(workspacePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("snippets.html"), struct {
		Languages []languageCategory
		DirName   string
	}{languageList, dirName})
}

func (s *Server) handleWorkspaceSnippetDetail(w http.ResponseWriter, r *http.Request, workspacePath, snippetPath string) {
	if snippetPath == "" {
		dirName := filepath.Base(workspacePath)
		http.Redirect(w, r, "/workspaces/"+dirName+"/snippets", http.StatusSeeOther)
		return
	}

	parts := strings.Split(snippetPath, "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	language := parts[0]
	filename := parts[1]

	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	var content []byte
	var foundExt string
	extensions := []string{".go", ".html", ".css", ".js", ".ts", ".py", ".sh", ".yaml", ".yml", ".json", ".md", ".sql", ".txt", ""}

	for _, ext := range extensions {
		filePath := language + "/" + filename + ext
		var errRead error
		content, errRead = fs.ReadFile(snippetsFS, filePath)
		if errRead == nil {
			foundExt = ext
			break
		}
	}

	if content == nil {
		http.NotFound(w, r)
		return
	}

	fm := parseFrontmatterMap(content)
	name := fm["name"]
	if name == "" {
		name = filename
	}
	description := fm["description"]
	whenToUse := fm["when_to_use"]
	codeContent := stripFrontmatter(string(content))

	dirName := filepath.Base(workspacePath)
	data := struct {
		Name        string
		FileName    string
		Language    string
		Description string
		WhenToUse   string
		Content     string
		Extension   string
		DirName     string
	}{
		Name:        name,
		FileName:    filename,
		Language:    language,
		Description: description,
		WhenToUse:   whenToUse,
		Content:     codeContent,
		Extension:   strings.TrimPrefix(foundExt, "."),
		DirName:     dirName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("snippet_detail.html"), data)
}

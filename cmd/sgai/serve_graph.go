package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

var (
	tmplNoWorkflowSVG *template.Template
	tmplFallbackSVG   *template.Template
)

func initGraphTemplates() {
	tmplNoWorkflowSVG = template.Must(template.New("noWorkflowSVG").Parse(
		`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="40" viewBox="0 0 200 40">
<rect width="100%" height="100%" fill="#f8fafc"/>
<text x="100" y="24" font-family="system-ui, sans-serif" font-size="12" fill="#64748b" text-anchor="middle">No workflow available</text>
</svg>`))

	tmplFallbackSVG = template.Must(template.New("fallbackSVG").Parse(
		`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="{{.Height}}" viewBox="0 0 400 {{.Height}}">
<rect width="100%" height="100%" fill="#f8fafc"/>
<text x="10" y="20" font-family="monospace" font-size="12" fill="#475569">{{range .Lines}}<tspan x="10" dy="{{.DY}}">{{.Text}}</tspan>{{end}}</text>
</svg>`))
}

func (s *Server) serveWorkflowSVG(w http.ResponseWriter, r *http.Request) {
	dirParam := r.URL.Query().Get("dir")
	if dirParam == "" {
		http.Error(w, "No directory specified", http.StatusBadRequest)
		return
	}

	dir, err := s.validateDirectory(dirParam)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusForbidden)
		return
	}

	wfState, _ := state.Load(statePath(dir))

	svgContent := getWorkflowSVG(dir, wfState.CurrentAgent)
	w.Header().Set("Content-Type", "image/svg+xml")
	if svgContent == "" {
		if err := tmplNoWorkflowSVG.Execute(w, nil); err != nil {
			log.Println("template execution failed:", err)
		}
		return
	}

	if _, err := w.Write([]byte(svgContent)); err != nil {
		log.Println("write failed:", err)
	}
}

func renderDotToSVG(dotContent string) string {
	dotPath, err := exec.LookPath("dot")
	if err != nil {
		return renderDotAsFallbackSVG(dotContent)
	}

	cmd := exec.Command(dotPath, "-Tsvg")
	cmd.Stdin = strings.NewReader(dotContent)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return renderDotAsFallbackSVG(dotContent)
	}
	return out.String()
}

func renderDotAsFallbackSVG(dotContent string) string {
	lines := linesWithTrailingEmpty(dotContent)
	height := max(20+len(lines)*16, 100)

	type lineData struct {
		DY   int
		Text string
	}
	var lineItems []lineData
	for i, line := range lines {
		dy := 16
		if i == 0 {
			dy = 0
		}
		lineItems = append(lineItems, lineData{DY: dy, Text: line})
	}

	var buf bytes.Buffer
	data := struct {
		Height int
		Lines  []lineData
	}{
		Height: height,
		Lines:  lineItems,
	}
	if err := tmplFallbackSVG.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

func getWorkflowSVGHash(dir string, currentAgent string) string {
	svg := getWorkflowSVG(dir, currentAgent)
	if svg == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(svg))
	return hex.EncodeToString(hash[:8])
}

func getWorkflowSVG(dir string, currentAgent string) string {
	goalPath := filepath.Join(dir, "GOAL.md")
	goalData, err := os.ReadFile(goalPath)
	if err != nil {
		return ""
	}

	metadata, err := parseYAMLFrontmatter(goalData)
	if err != nil {
		return ""
	}

	d, err := parseFlow(metadata.Flow, dir)
	if err != nil {
		return ""
	}

	dotContent := d.toDOT()

	if currentAgent != "" {
		dotContent = injectCurrentAgentStyle(dotContent, currentAgent)
	}
	dotContent = injectLightTheme(dotContent)

	return renderDotToSVG(dotContent)
}

func injectCurrentAgentStyle(dot, currentAgent string) string {
	agentLine := fmt.Sprintf(`    "%s"`, currentAgent)
	styledLine := fmt.Sprintf(`    "%s" [style=filled, fillcolor="#10b981", fontcolor=white]`, currentAgent)

	if !strings.Contains(dot, agentLine) {
		return dot
	}

	return strings.Replace(dot, agentLine, styledLine, 1)
}

func injectLightTheme(dot string) string {
	lightTheme := `    bgcolor="transparent"
    node [style=filled, fillcolor="#e2e8f0", fontcolor="#1e293b", color="#94a3b8"]
    edge [color="#64748b", fontcolor="#475569"]`

	braceIdx := strings.Index(dot, "{")
	if braceIdx == -1 {
		return dot
	}

	return dot[:braceIdx+1] + "\n" + lightTheme + dot[braceIdx+1:]
}

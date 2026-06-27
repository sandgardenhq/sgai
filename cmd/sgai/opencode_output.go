package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

func formatLogTimestamp(t time.Time) string {
	return "[" + t.UTC().Format(time.RFC3339) + "]"
}

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	lines := linesWithTrailingEmpty(string(data))
	for i, line := range lines {
		if i < len(lines)-1 || line != "" {
			timestamp := formatLogTimestamp(time.Now())
			if _, errWrite := p.w.Write([]byte(timestamp + p.prefix + line + "\n")); errWrite != nil {
				return 0, errWrite
			}
		}
	}
	return len(data), nil
}

type sessionIDCaptureWriter struct {
	pending        []byte
	sessionID      string
	detectedWriter io.Writer
	passthrough    io.Writer
}

func (w *sessionIDCaptureWriter) Write(data []byte) (int, error) {
	w.pending = append(w.pending, data...)
	if errProcess := w.processLines(false); errProcess != nil {
		return 0, errProcess
	}
	return len(data), nil
}

func (w *sessionIDCaptureWriter) Flush() {
	_ = w.processLines(true)
}

func (w *sessionIDCaptureWriter) processLines(flush bool) error {
	for {
		idx := bytes.IndexByte(w.pending, '\n')
		if idx == -1 {
			break
		}
		line := string(w.pending[:idx])
		w.pending = w.pending[idx+1:]
		if sessionID, agent := sessionIDFromPluginLine(line); sessionID != "" {
			if w.sessionID == "" {
				w.sessionID = sessionID
			}
			if errDetect := w.printDetectedSessionID(sessionID, agent); errDetect != nil {
				return errDetect
			}
			continue
		}
		if errWrite := w.writePassthrough(line + "\n"); errWrite != nil {
			return errWrite
		}
	}
	if flush && len(w.pending) > 0 {
		line := string(w.pending)
		w.pending = nil
		if sessionID, agent := sessionIDFromPluginLine(line); sessionID != "" {
			if w.sessionID == "" {
				w.sessionID = sessionID
			}
			return w.printDetectedSessionID(sessionID, agent)
		}
		return w.writePassthrough(line)
	}
	return nil
}

func (w *sessionIDCaptureWriter) printDetectedSessionID(sessionID, agent string) error {
	if w.detectedWriter == nil {
		return nil
	}
	message := "Detected sessionID: " + sessionID
	if agent != "" {
		message = "Detected sessionID for " + agent + ": " + sessionID
	}
	_, errWrite := fmt.Fprintln(w.detectedWriter, message)
	return errWrite
}

func (w *sessionIDCaptureWriter) writePassthrough(data string) error {
	if w.passthrough == nil || data == "" {
		return nil
	}
	_, errWrite := io.WriteString(w.passthrough, data)
	return errWrite
}

func sessionIDFromPluginLine(line string) (string, string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		return "", ""
	}
	var values map[string]string
	if errUnmarshal := json.Unmarshal([]byte(line), &values); errUnmarshal != nil {
		return "", ""
	}
	sessionID := strings.TrimSpace(values["sessionID"])
	agent := strings.TrimSpace(values["agent"])
	if sessionID == "" || strings.ContainsAny(sessionID, `"{}`) {
		return "", ""
	}
	if len(values) == 1 {
		return sessionID, ""
	}
	if len(values) == 2 && agent != "" {
		return sessionID, agent
	}
	return "", ""
}

type part struct {
	ID        string         `json:"id,omitempty"`
	CallID    string         `json:"callID,omitempty"`
	SessionID string         `json:"sessionID,omitempty"`
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	Model     string         `json:"model,omitempty"`
	State     *toolState     `json:"state,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Cost      float64        `json:"cost,omitempty"`
	Tokens    partTokens     `json:"tokens"`
}

type partTokens struct {
	Input     int        `json:"input"`
	Output    int        `json:"output"`
	Reasoning int        `json:"reasoning"`
	Cache     cacheStats `json:"cache"`
}

type cacheStats struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

type toolState struct {
	Status   string         `json:"status"`
	Input    map[string]any `json:"input"`
	Title    string         `json:"title,omitempty"`
	Output   toolOutput     `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type toolOutput struct {
	raw json.RawMessage
}

func (o *toolOutput) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		o.raw = nil
		return nil
	}
	if !json.Valid(data) {
		return fmt.Errorf("invalid tool output json")
	}
	o.raw = append(o.raw[:0], data...)
	return nil
}

func (o toolOutput) MarshalJSON() ([]byte, error) {
	if len(o.raw) == 0 {
		return []byte("null"), nil
	}
	return o.raw.MarshalJSON()
}

func (o toolOutput) String() string {
	if len(o.raw) == 0 {
		return ""
	}
	var text string
	if errUnmarshal := json.Unmarshal(o.raw, &text); errUnmarshal == nil {
		return text
	}
	var buf bytes.Buffer
	if errCompact := json.Compact(&buf, o.raw); errCompact == nil {
		return buf.String()
	}
	return strings.TrimSpace(string(o.raw))
}

func (o toolOutput) sessionIDs() []string {
	if len(o.raw) == 0 {
		return nil
	}
	var value any
	if errUnmarshal := json.Unmarshal(o.raw, &value); errUnmarshal != nil {
		return nil
	}
	var sessionIDs []string
	collectToolOutputSessionIDs(value, &sessionIDs)
	return sessionIDs
}

func collectToolOutputSessionIDs(value any, sessionIDs *[]string) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectToolOutputSessionIDs(item, sessionIDs)
		}
	case map[string]any:
		for key, item := range typed {
			switch normalizedToolOutputKey(key) {
			case "sessionid", "session":
				addStringValue(sessionIDs, stringValue(item))
			}
			collectToolOutputSessionIDs(item, sessionIDs)
		}
	}
}

func addStringValue(values *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" || slices.Contains(*values, value) {
		return
	}
	*values = append(*values, value)
}

func isTaskToolName(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "task", "tasks":
		return true
	default:
		return false
	}
}

func normalizedToolOutputKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	return key
}

func todoStatusSymbol(status string) string {
	switch status {
	case "pending":
		return "○"
	case "in_progress":
		return "◐"
	case "completed":
		return "●"
	case "cancelled":
		return "✕"
	default:
		return "○"
	}
}

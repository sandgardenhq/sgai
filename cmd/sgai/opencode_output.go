package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

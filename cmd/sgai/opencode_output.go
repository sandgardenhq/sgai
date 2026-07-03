package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const ansiEscapePattern = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiEscapeRE = regexp.MustCompile(ansiEscapePattern)
var ansiOSCRe = regexp.MustCompile("[\u001B\u009B]].*?(?:\u0007|\u001B\\\\)")

type ansiStrippingWriter struct {
	w       io.Writer
	pending []byte
}

func (w *ansiStrippingWriter) Write(data []byte) (int, error) {
	if w.w == nil {
		return len(data), nil
	}
	stripped := w.strip(data)
	written, errWrite := w.w.Write(stripped)
	if errWrite != nil {
		return 0, errWrite
	}
	if written != len(stripped) {
		return 0, io.ErrShortWrite
	}
	return len(data), nil
}

func (w *ansiStrippingWriter) strip(data []byte) []byte {
	input := w.pending
	input = append(input, data...)
	w.pending = nil
	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); {
		b := input[i]
		switch b {
		case 0x1b:
			next, ok := skipEscapeSequence(input, i)
			if !ok {
				w.pending = append(w.pending, input[i:]...)
				return out
			}
			i = next
		case 0x9b:
			next, ok := skipCSISequence(input, i+1)
			if !ok {
				w.pending = append(w.pending, input[i:]...)
				return out
			}
			i = next
		case 0x9d:
			next, ok := skipOSCSequence(input, i+1)
			if !ok {
				w.pending = append(w.pending, input[i:]...)
				return out
			}
			i = next
		default:
			out = append(out, b)
			i++
		}
	}
	return out
}

func skipEscapeSequence(data []byte, start int) (int, bool) {
	if start+1 >= len(data) {
		return start, false
	}
	switch data[start+1] {
	case '[':
		return skipCSISequence(data, start+2)
	case ']':
		return skipOSCSequence(data, start+2)
	default:
		return skipNonCSISequence(data, start+1)
	}
}

func skipNonCSISequence(data []byte, start int) (int, bool) {
	for i := start; i < len(data); i++ {
		if isANSIPrefixByte(data[i]) {
			continue
		}
		return i + 1, true
	}
	return start, false
}

func isANSIPrefixByte(b byte) bool {
	switch b {
	case '[', '\\', ']', '(', ')', '#', ';', '?':
		return true
	default:
		return false
	}
}

func skipCSISequence(data []byte, start int) (int, bool) {
	for i := start; i < len(data); i++ {
		if data[i] >= 0x40 && data[i] <= 0x7e {
			return i + 1, true
		}
	}
	return start, false
}

func skipOSCSequence(data []byte, start int) (int, bool) {
	for i := start; i < len(data); i++ {
		switch data[i] {
		case 0x07:
			return i + 1, true
		case 0x1b:
			if i+1 >= len(data) {
				return start, false
			}
			if data[i+1] == '\\' {
				return i + 2, true
			}
		}
	}
	return start, false
}

func stripANSIBytes(data []byte) []byte {
	return ansiEscapeRE.ReplaceAll(ansiOSCRe.ReplaceAll(data, nil), nil)
}

func formatLogTimestamp(t time.Time) string {
	return "[" + t.UTC().Format(time.RFC3339) + "]"
}

type prefixWriter struct {
	prefix  string
	w       io.Writer
	pending []byte
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	p.pending = append(p.pending, data...)
	if errProcess := p.processLines(false); errProcess != nil {
		return 0, errProcess
	}
	return len(data), nil
}

func (p *prefixWriter) Flush() error {
	return p.processLines(true)
}

func (p *prefixWriter) processLines(flush bool) error {
	for {
		idx := bytes.IndexByte(p.pending, '\n')
		if idx == -1 {
			break
		}
		line := string(p.pending[:idx])
		p.pending = p.pending[idx+1:]
		if errWrite := p.writePrefixedLine(line); errWrite != nil {
			return errWrite
		}
	}
	if flush && len(p.pending) > 0 {
		line := string(p.pending)
		p.pending = nil
		return p.writePrefixedLine(line)
	}
	return nil
}

func (p *prefixWriter) writePrefixedLine(line string) error {
	timestamp := formatLogTimestamp(time.Now())
	_, errWrite := p.w.Write([]byte(timestamp + p.prefix + line + "\n"))
	return errWrite
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

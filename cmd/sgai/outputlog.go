package main

import (
	"container/ring"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	outputBufferSize = 100
)

type logLine struct {
	prefix string
	text   string
}

type circularLogBuffer struct {
	mu   sync.RWMutex
	ring *ring.Ring
	size int
}

func newCircularLogBuffer() *circularLogBuffer {
	return &circularLogBuffer{
		ring: ring.New(outputBufferSize),
	}
}

func (c *circularLogBuffer) add(line logLine) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ring.Value = line
	c.ring = c.ring.Next()
	if c.size < outputBufferSize {
		c.size++
	}
}

func (c *circularLogBuffer) lines() []logLine {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.size == 0 {
		return nil
	}

	result := make([]logLine, 0, c.size)
	startRing := c.ring
	if c.size < outputBufferSize {
		startRing = c.ring.Move(-c.size)
	}

	startRing.Do(func(v any) {
		if v != nil {
			result = append(result, v.(logLine))
		}
	})

	return result[:c.size]
}

type ringWriter struct {
	mu      sync.Mutex
	ring    *ring.Ring
	size    int
	partial []byte
}

func newRingWriter() *ringWriter {
	return &ringWriter{
		ring: ring.New(outputBufferSize),
	}
}

func (r *ringWriter) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(data)
	combined := r.partial
	combined = append(combined, data...)
	lines := splitLines(combined)

	for i := 0; i < len(lines)-1; i++ {
		r.addLine(lines[i])
	}

	if len(combined) > 0 && combined[len(combined)-1] == '\n' {
		r.addLine(lines[len(lines)-1])
		r.partial = nil
	} else {
		r.partial = []byte(lines[len(lines)-1])
	}

	return n, nil
}

func (r *ringWriter) dump(w io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 && len(r.partial) == 0 {
		return
	}

	startRing := r.ring
	if r.size < outputBufferSize {
		startRing = r.ring.Move(-r.size)
	}

	startRing.Do(func(v any) {
		if v != nil {
			if _, err := fmt.Fprintln(w, v.(string)); err != nil {
				log.Println("write failed:", err)
			}
		}
	})

	if len(r.partial) > 0 {
		if _, err := fmt.Fprintln(w, string(r.partial)); err != nil {
			log.Println("write failed:", err)
		}
	}
}

func (r *ringWriter) addLine(line string) {
	r.ring.Value = line
	r.ring = r.ring.Next()
	if r.size < outputBufferSize {
		r.size++
	}
}

func splitLines(data []byte) []string {
	return strings.Split(string(data), "\n")
}

func prepareLogFile(logPath string) (*os.File, error) {
	if err := rotateLogFile(logPath); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	return f, nil
}

func rotateLogFile(logPath string) error {
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(logPath, logPath+".old")
}

type sessionLogWriter struct {
	mu            sync.Mutex
	partial       []byte
	sess          *session
	workspacePath string
	srv           *Server
	workspaceName string
}

func newSessionLogWriter(sess *session, workspacePath string, srv *Server, workspaceName string) *sessionLogWriter {
	return &sessionLogWriter{
		sess:          sess,
		workspacePath: workspacePath,
		srv:           srv,
		workspaceName: workspaceName,
	}
}

func (w *sessionLogWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(data)
	combined := w.partial
	combined = append(combined, data...)
	lines := splitLines(combined)

	for i := 0; i < len(lines)-1; i++ {
		w.addLine(lines[i])
	}

	if len(combined) > 0 && combined[len(combined)-1] == '\n' {
		w.addLine(lines[len(lines)-1])
		w.partial = nil
	} else {
		w.partial = []byte(lines[len(lines)-1])
	}

	return n, nil
}

func (w *sessionLogWriter) addLine(text string) {
	w.sess.outputLog.add(logLine{text: text})
	w.srv.notifyStateChange()
}

func buildAgentStderrWriter(logWriter, stderrLog io.Writer) io.Writer {
	base := io.Writer(os.Stderr)
	if logWriter != nil && stderrLog != nil {
		return io.MultiWriter(base, logWriter, stderrLog)
	}
	if logWriter != nil {
		return io.MultiWriter(base, logWriter)
	}
	if stderrLog != nil {
		return io.MultiWriter(base, stderrLog)
	}
	return base
}

func buildAgentStdoutWriter(logWriter, stdoutLog io.Writer) io.Writer {
	base := io.Writer(os.Stdout)
	if logWriter != nil && stdoutLog != nil {
		return io.MultiWriter(base, logWriter, stdoutLog)
	}
	if logWriter != nil {
		return io.MultiWriter(base, logWriter)
	}
	if stdoutLog != nil {
		return io.MultiWriter(base, stdoutLog)
	}
	return base
}

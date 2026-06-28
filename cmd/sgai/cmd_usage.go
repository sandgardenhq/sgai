package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func cmdTokenStats(args []string) {
	fs := flag.NewFlagSet("token-stats", flag.ExitOnError)
	listenAddr := fs.String("listen-addr", "127.0.0.1:0", "listen address (unused)")
	_ = listenAddr
	fs.Usage = func() {
		fmt.Println("sgai token-stats <workspace-path>")
		fmt.Println("")
		fmt.Println("Reads .sgai/sessions.jsonl in the workspace and aggregates token usage")
		fmt.Println("from the opencode database, broken down by agent and model.")
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}
	workspacePath := fs.Arg(0)
	absWorkspace, errAbs := filepath.Abs(workspacePath)
	if errAbs == nil {
		workspacePath = absWorkspace
	}

	sessionIDs, errSessions := readSessionsJSONL(filepath.Join(workspacePath, ".sgai", "sessions.jsonl"))
	if errSessions != nil {
		log.Fatalln("cannot read sessions.jsonl:", errSessions)
	}
	if len(sessionIDs) == 0 {
		fmt.Println("no sessions recorded in", filepath.Join(workspacePath, ".sgai", "sessions.jsonl"))
		return
	}

	dbPath := resolveOpencodeDBPath()
	usage, errQuery := queryTokenUsage(dbPath, sessionIDs)
	if errQuery != nil {
		log.Fatalln("cannot query opencode database:", errQuery)
	}
	printTokenUsage(usage)
}

type sessionEntry struct {
	SessionID string `json:"sessionID"`
	Agent     string `json:"agent"`
}

func readSessionsJSONL(path string) ([]string, error) {
	file, errOpen := os.Open(path)
	if errOpen != nil {
		return nil, fmt.Errorf("opening sessions file: %w", errOpen)
	}
	defer func() {
		if errClose := file.Close(); errClose != nil {
			log.Println("failed to close sessions file:", errClose)
		}
	}()
	var sessionIDs []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry sessionEntry
		if errJSON := json.Unmarshal([]byte(line), &entry); errJSON != nil {
			continue
		}
		if entry.SessionID != "" {
			sessionIDs = append(sessionIDs, entry.SessionID)
		}
	}
	if errScan := scanner.Err(); errScan != nil {
		return nil, fmt.Errorf("reading sessions file: %w", errScan)
	}
	return sessionIDs, nil
}

func resolveOpencodeDBPath() string {
	if envPath := strings.TrimSpace(os.Getenv("OPENCODE_DB")); envPath != "" && envPath != ":memory:" {
		if filepath.IsAbs(envPath) {
			return envPath
		}
		dataDir := resolveOpencodeDataDir()
		return filepath.Join(dataDir, envPath)
	}
	dataDir := resolveOpencodeDataDir()
	return filepath.Join(dataDir, "opencode.db")
}

func resolveOpencodeDataDir() string {
	cmd := exec.Command("opencode", "debug", "paths")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if errRun := cmd.Run(); errRun != nil {
		log.Println("failed to run 'opencode debug paths', falling back to XDG default:", errRun)
		home, errHome := os.UserHomeDir()
		if errHome != nil {
			log.Fatalln("cannot determine home directory:", errHome)
		}
		return filepath.Join(home, ".local", "share", "opencode")
	}
	for line := range strings.SplitSeq(stdout.String(), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), " ")
		if found && strings.TrimSpace(key) == "data" {
			return strings.TrimSpace(value)
		}
	}
	home, errHome := os.UserHomeDir()
	if errHome != nil {
		log.Fatalln("cannot determine home directory:", errHome)
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

type tokenUsageRow struct {
	Agent            string
	Model            string
	Input            int64
	Output           int64
	CacheRead        int64
	CacheWrite       int64
	Reasoning        int64
	Other            int64
	Total            int64
	SessionCount     int64
}

type tokenUsage struct {
	Rows   []tokenUsageRow
	Totals tokenUsageRow
}

func queryTokenUsage(dbPath string, sessionIDs []string) (tokenUsage, error) {
	dsn := "file:" + dbPath + "?mode=ro&_pragma=busy_timeout(5000)"
	placeholders := make([]any, len(sessionIDs))
	placeList := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		placeholders[i] = id
		placeList[i] = "?"
	}
	query := `
		SELECT
			COALESCE(agent, '') AS agent,
			COALESCE(model, '') AS model,
			SUM(tokens_input)       AS input,
			SUM(tokens_output)      AS output,
			SUM(tokens_cache_read)  AS cache_read,
			SUM(tokens_cache_write) AS cache_write,
			SUM(tokens_reasoning)   AS reasoning,
			COUNT(*)                AS sessions
		FROM session
		WHERE id IN (` + strings.Join(placeList, ", ") + `)
		GROUP BY agent, model
		ORDER BY agent, model
	`
	var rows *sql.Rows
	var errQuery error
	for attempt := 0; attempt < 5; attempt++ {
		db, errOpen := sql.Open("sqlite", dsn)
		if errOpen != nil {
			return tokenUsage{}, fmt.Errorf("opening opencode database: %w", errOpen)
		}
		rows, errQuery = db.Query(query, placeholders...)
		if errQuery == nil {
			defer func() {
				if errClose := rows.Close(); errClose != nil {
					log.Println("failed to close usage rows:", errClose)
				}
				if errClose := db.Close(); errClose != nil {
					log.Println("failed to close opencode database:", errClose)
				}
			}()
			break
		}
		if errClose := db.Close(); errClose != nil {
			log.Println("failed to close opencode database after error:", errClose)
		}
		log.Println("opencode database unavailable, retrying in 1s:", errQuery)
		time.Sleep(time.Second)
	}
	if errQuery != nil {
		return tokenUsage{}, fmt.Errorf("querying token usage: %w", errQuery)
	}

	var usage tokenUsage
	for rows.Next() {
		var r tokenUsageRow
		if errScan := rows.Scan(&r.Agent, &r.Model, &r.Input, &r.Output, &r.CacheRead, &r.CacheWrite, &r.Reasoning, &r.SessionCount); errScan != nil {
			return tokenUsage{}, fmt.Errorf("scanning usage row: %w", errScan)
		}
		r.Other = r.Reasoning
		r.Total = r.Input + r.Output + r.CacheRead + r.CacheWrite + r.Reasoning
		usage.Rows = append(usage.Rows, r)
		usage.Totals.Input += r.Input
		usage.Totals.Output += r.Output
		usage.Totals.CacheRead += r.CacheRead
		usage.Totals.CacheWrite += r.CacheWrite
		usage.Totals.Reasoning += r.Reasoning
		usage.Totals.Other += r.Other
		usage.Totals.Total += r.Total
		usage.Totals.SessionCount += r.SessionCount
	}
	if errRows := rows.Err(); errRows != nil {
		return tokenUsage{}, fmt.Errorf("reading usage rows: %w", errRows)
	}
	return usage, nil
}

func printTokenUsage(usage tokenUsage) {
	if len(usage.Rows) == 0 {
		fmt.Println("no token usage found for the recorded sessions")
		return
	}
	fmt.Printf("%-20s %-45s %12s %12s %14s %15s %12s %12s %15s %10s\n",
		"AGENT", "MODEL", "INPUT", "OUTPUT", "CACHED INPUT", "CACHED OUTPUT", "OTHER", "REASONING", "TOTAL", "SESSIONS")
	fmt.Println(strings.Repeat("-", 168))
	for _, r := range usage.Rows {
		fmt.Printf("%-20s %-45s %12d %12d %14d %15d %12d %12d %15d %10d\n",
			r.Agent, modelDisplay(r.Model), r.Input, r.Output, r.CacheRead, r.CacheWrite, r.Other, r.Reasoning, r.Total, r.SessionCount)
	}
	fmt.Println(strings.Repeat("-", 168))
	t := usage.Totals
	fmt.Printf("%-20s %-45s %12d %12d %14d %15d %12d %12d %15d %10d\n",
		"TOTAL", "", t.Input, t.Output, t.CacheRead, t.CacheWrite, t.Other, t.Reasoning, t.Total, t.SessionCount)
}

type modelDescriptor struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant"`
}

func modelDisplay(raw string) string {
	if raw == "" {
		return ""
	}
	var desc modelDescriptor
	if errJSON := json.Unmarshal([]byte(raw), &desc); errJSON != nil {
		return raw
	}
	parts := []string{}
	if desc.ProviderID != "" {
		parts = append(parts, desc.ProviderID)
	}
	if desc.ID != "" {
		parts = append(parts, desc.ID)
	}
	if desc.Variant != "" && desc.Variant != "default" {
		parts = append(parts, desc.Variant)
	}
	return strings.Join(parts, "/")
}

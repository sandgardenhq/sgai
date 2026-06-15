package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	_ "modernc.org/sqlite"
)

const usageStateMaxBytes = 10 * 1024 * 1024

var userConfigDir = os.UserConfigDir

type usageStore struct {
	db *sql.DB
}

type usageWorkspaceContext struct {
	WorkspacePath     string
	WorkspaceName     string
	RootWorkspacePath string
	RootWorkspaceName string
}

type usageQuery struct {
	From        time.Time
	To          time.Time
	Project     string
	RootProject string
}

type usageTotals struct {
	Cost                       float64          `json:"cost"`
	MeteredReportedCost        float64          `json:"meteredReportedCost"`
	APIEquivalentCost          float64          `json:"apiEquivalentCost"`
	APIEquivalentCostAvailable bool             `json:"apiEquivalentCostAvailable"`
	Tokens                     state.TokenUsage `json:"tokens"`
}

type usageDailyPoint struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

type usageRow struct {
	Date                       string           `json:"date"`
	Project                    string           `json:"project"`
	RootProject                string           `json:"rootProject"`
	WorkspacePath              string           `json:"workspacePath"`
	RootWorkspacePath          string           `json:"rootWorkspacePath"`
	Source                     string           `json:"source"`
	Cost                       float64          `json:"cost"`
	MeteredReportedCost        float64          `json:"meteredReportedCost"`
	APIEquivalentCost          float64          `json:"apiEquivalentCost"`
	APIEquivalentCostAvailable bool             `json:"apiEquivalentCostAvailable"`
	Tokens                     state.TokenUsage `json:"tokens"`
}

type usageFilters struct {
	Projects     []string `json:"projects"`
	RootProjects []string `json:"rootProjects"`
}

type usageResponse struct {
	Totals  usageTotals       `json:"totals"`
	Daily   []usageDailyPoint `json:"daily"`
	Rows    []usageRow        `json:"rows"`
	Filters usageFilters      `json:"filters"`
	Warning string            `json:"warning,omitempty"`
}

func globalUsageDBPath(configDirFunc func() (string, error)) (string, error) {
	dir, errConfigDir := configDirFunc()
	if errConfigDir != nil {
		return "", fmt.Errorf("resolving user config directory: %w", errConfigDir)
	}
	return filepath.Join(dir, "sgai", "usage.sqlite"), nil
}

func openGlobalUsageStore() (*usageStore, error) {
	path, errPath := globalUsageDBPath(userConfigDir)
	if errPath != nil {
		return nil, errPath
	}
	if errMkdir := os.MkdirAll(filepath.Dir(path), 0o700); errMkdir != nil {
		return nil, fmt.Errorf("creating usage database directory: %w", errMkdir)
	}
	return openUsageStore(path)
}

func openUsageStore(path string) (*usageStore, error) {
	db, errOpen := sql.Open("sqlite", path)
	if errOpen != nil {
		return nil, fmt.Errorf("opening usage database: %w", errOpen)
	}
	if errPing := db.Ping(); errPing != nil {
		if errClose := db.Close(); errClose != nil {
			log.Println("failed to close usage database after open error:", errClose)
		}
		return nil, fmt.Errorf("opening usage database: %w", errPing)
	}
	store := &usageStore{db: db}
	if errMigrate := store.migrate(); errMigrate != nil {
		if errClose := db.Close(); errClose != nil {
			log.Println("failed to close usage database after migration error:", errClose)
		}
		return nil, fmt.Errorf("migrating usage database: %w", errMigrate)
	}
	return store, nil
}

func (s *usageStore) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *usageStore) migrate() error {
	_, errExec := s.db.Exec(`CREATE TABLE IF NOT EXISTS usage_steps (
		source TEXT NOT NULL,
		session_id TEXT NOT NULL,
		step_id TEXT NOT NULL,
		step_index INTEGER NOT NULL,
		workspace_path TEXT NOT NULL,
		workspace_name TEXT NOT NULL,
		root_workspace_path TEXT NOT NULL,
		root_workspace_name TEXT NOT NULL,
		agent TEXT NOT NULL,
		model TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		cost REAL NOT NULL,
		metered_reported_cost REAL NOT NULL,
		api_equivalent_cost REAL NOT NULL,
		api_equivalent_cost_available INTEGER NOT NULL,
		input_tokens INTEGER NOT NULL,
		output_tokens INTEGER NOT NULL,
		reasoning_tokens INTEGER NOT NULL,
		cache_read_tokens INTEGER NOT NULL,
		cache_write_tokens INTEGER NOT NULL,
		PRIMARY KEY (source, session_id, step_id, step_index, agent)
	)`)
	return errExec
}

func (s *usageStore) replaceSessionUsage(source string, ctx usageWorkspaceContext, sessions []state.SessionUsage) error {
	return s.replaceUsage(source, ctx, sessions, func(tx *sql.Tx) error {
		for _, session := range sessions {
			if _, errDelete := tx.Exec(`DELETE FROM usage_steps WHERE source = ? AND session_id = ?`, source, session.SessionID); errDelete != nil {
				return fmt.Errorf("deleting previous usage rows: %w", errDelete)
			}
		}
		return nil
	})
}

func (s *usageStore) replaceWorkspaceUsage(ctx usageWorkspaceContext, sessions []state.SessionUsage) error {
	source := "backfill"
	return s.replaceUsage(source, ctx, sessions, func(tx *sql.Tx) error {
		if _, errDelete := tx.Exec(`DELETE FROM usage_steps WHERE source = ? AND workspace_path = ?`, source, ctx.WorkspacePath); errDelete != nil {
			return fmt.Errorf("deleting previous workspace usage rows: %w", errDelete)
		}
		return nil
	})
}

func (s *usageStore) replaceUsage(source string, ctx usageWorkspaceContext, sessions []state.SessionUsage, deletePrevious func(*sql.Tx) error) error {
	if s == nil || s.db == nil {
		return nil
	}
	tx, errBegin := s.db.BeginTx(context.Background(), nil)
	if errBegin != nil {
		return fmt.Errorf("beginning usage transaction: %w", errBegin)
	}
	defer func() {
		if errRollback := tx.Rollback(); errRollback != nil && errRollback != sql.ErrTxDone {
			log.Println("failed to roll back usage transaction:", errRollback)
		}
	}()

	if errDelete := deletePrevious(tx); errDelete != nil {
		return errDelete
	}

	stmt, errPrepare := tx.Prepare(`INSERT INTO usage_steps (
		source, session_id, step_id, step_index, workspace_path, workspace_name,
		root_workspace_path, root_workspace_name, agent, model, timestamp, cost,
		metered_reported_cost, api_equivalent_cost, api_equivalent_cost_available,
		input_tokens, output_tokens, reasoning_tokens, cache_read_tokens, cache_write_tokens
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if errPrepare != nil {
		return fmt.Errorf("preparing usage insert: %w", errPrepare)
	}
	defer func() {
		if errClose := stmt.Close(); errClose != nil {
			log.Println("failed to close usage statement:", errClose)
		}
	}()

	for _, session := range sessions {
		steps := usageSessionSteps(session)
		for index, step := range steps {
			stepID := step.StepID
			if stepID == "" {
				stepID = fmt.Sprintf("%s-step-%d", session.SessionID, index+1)
			}
			agent := step.Agent
			if agent == "" {
				agent = session.Agent
			}
			model := session.Model
			if _, errExec := stmt.Exec(source, session.SessionID, stepID, index, ctx.WorkspacePath, ctx.WorkspaceName, ctx.RootWorkspacePath, ctx.RootWorkspaceName, agent, model, normalizeStepTimestamp(step.Timestamp), step.Cost, step.MeteredReportedCost, step.APIEquivalentCost, boolInt(step.APIEquivalentCostAvailable), step.Tokens.Input, step.Tokens.Output, step.Tokens.Reasoning, step.Tokens.CacheRead, step.Tokens.CacheWrite); errExec != nil {
				return fmt.Errorf("inserting usage row: %w", errExec)
			}
		}
	}

	if errCommit := tx.Commit(); errCommit != nil {
		return fmt.Errorf("committing usage transaction: %w", errCommit)
	}
	return nil
}

func usageSessionSteps(session state.SessionUsage) []state.StepCost {
	if len(session.Steps) > 0 {
		return session.Steps
	}
	if !hasSessionUsage(session) {
		return nil
	}
	return []state.StepCost{{
		StepID:                     session.SessionID + "-summary",
		Agent:                      session.Agent,
		SessionID:                  session.SessionID,
		Cost:                       sessionSummaryCost(session),
		MeteredReportedCost:        session.MeteredReportedCost,
		APIEquivalentCost:          session.APIEquivalentCost,
		APIEquivalentCostAvailable: session.APIEquivalentCostAvailable,
		Tokens:                     session.Tokens,
	}}
}

func hasSessionUsage(session state.SessionUsage) bool {
	return session.MeteredReportedCost != 0 || session.APIEquivalentCost != 0 || tokenUsageTotal(session.Tokens) != 0
}

func sessionSummaryCost(session state.SessionUsage) float64 {
	if session.APIEquivalentCostAvailable {
		return session.APIEquivalentCost
	}
	return session.MeteredReportedCost
}

func tokenUsageTotal(tokens state.TokenUsage) int {
	return tokens.Input + tokens.Output + tokens.Reasoning + tokens.CacheRead + tokens.CacheWrite
}

func (s *usageStore) query(q usageQuery) (usageResponse, error) {
	resp := emptyUsageResponse()
	filters, errFilters := s.queryFilters()
	if errFilters != nil {
		return resp, errFilters
	}
	resp.Filters = filters

	where, args := usageWhere(q)
	rows, errRows := s.db.Query(`SELECT substr(timestamp, 1, 10), workspace_name, root_workspace_name, workspace_path, root_workspace_path, GROUP_CONCAT(DISTINCT source),
		SUM(cost), SUM(metered_reported_cost), SUM(api_equivalent_cost), MAX(api_equivalent_cost_available),
		SUM(input_tokens), SUM(output_tokens), SUM(reasoning_tokens), SUM(cache_read_tokens), SUM(cache_write_tokens)
		FROM `+effectiveUsageStepsQuery()+` `+where+` GROUP BY substr(timestamp, 1, 10), workspace_name, root_workspace_name, workspace_path, root_workspace_path ORDER BY 1, 3, 2`, args...)
	if errRows != nil {
		return resp, fmt.Errorf("querying usage rows: %w", errRows)
	}
	defer func() {
		if errClose := rows.Close(); errClose != nil {
			log.Println("failed to close usage rows:", errClose)
		}
	}()

	daily := map[string]float64{}
	for rows.Next() {
		var row usageRow
		var available int
		if errScan := rows.Scan(&row.Date, &row.Project, &row.RootProject, &row.WorkspacePath, &row.RootWorkspacePath, &row.Source, &row.Cost, &row.MeteredReportedCost, &row.APIEquivalentCost, &available, &row.Tokens.Input, &row.Tokens.Output, &row.Tokens.Reasoning, &row.Tokens.CacheRead, &row.Tokens.CacheWrite); errScan != nil {
			return resp, fmt.Errorf("scanning usage row: %w", errScan)
		}
		row.APIEquivalentCostAvailable = available != 0
		resp.Rows = append(resp.Rows, row)
		resp.Totals.Cost += row.Cost
		resp.Totals.MeteredReportedCost += row.MeteredReportedCost
		resp.Totals.APIEquivalentCost += row.APIEquivalentCost
		resp.Totals.APIEquivalentCostAvailable = resp.Totals.APIEquivalentCostAvailable || row.APIEquivalentCostAvailable
		resp.Totals.Tokens.Add(row.Tokens)
		daily[row.Date] += row.Cost
	}
	if errRows := rows.Err(); errRows != nil {
		return resp, fmt.Errorf("reading usage rows: %w", errRows)
	}
	for date, cost := range daily {
		resp.Daily = append(resp.Daily, usageDailyPoint{Date: date, Cost: cost})
	}
	slices.SortFunc(resp.Daily, func(a, b usageDailyPoint) int { return strings.Compare(a.Date, b.Date) })
	return resp, nil
}

func usageWhere(q usageQuery) (string, []any) {
	where := []string{"date(timestamp) >= date(?)", "date(timestamp) <= date(?)"}
	args := []any{q.From.Format(time.DateOnly), q.To.Format(time.DateOnly)}
	if q.Project != "" {
		where = append(where, "workspace_name = ?")
		args = append(args, q.Project)
	}
	if q.RootProject != "" {
		where = append(where, "root_workspace_name = ?")
		args = append(args, q.RootProject)
	}
	return "WHERE " + strings.Join(where, " AND "), args
}

func effectiveUsageStepsQuery() string {
	return `(
		SELECT * FROM usage_steps u
		WHERE NOT (
			u.source = 'backfill'
			AND EXISTS (
				SELECT 1 FROM usage_steps live
				WHERE live.source = 'session'
				AND live.workspace_path = u.workspace_path
				AND live.agent = u.agent
				AND (
					(
						live.session_id = u.session_id
						AND live.step_id = u.step_id
						AND live.step_index = u.step_index
					)
					OR (
						live.step_id = u.step_id
						AND live.timestamp = u.timestamp
						AND live.input_tokens = u.input_tokens
						AND live.output_tokens = u.output_tokens
						AND live.reasoning_tokens = u.reasoning_tokens
						AND live.cache_read_tokens = u.cache_read_tokens
						AND live.cache_write_tokens = u.cache_write_tokens
					)
				)
			)
		)
	) AS effective_usage_steps`
}

func (s *usageStore) queryFilters() (usageFilters, error) {
	var filters usageFilters
	projectRows, errProjects := s.db.Query(`SELECT DISTINCT workspace_name FROM ` + effectiveUsageStepsQuery() + ` WHERE workspace_name <> '' ORDER BY workspace_name`)
	if errProjects != nil {
		return filters, fmt.Errorf("querying usage project filters: %w", errProjects)
	}
	projects, errReadProjects := readStringRows(projectRows)
	if errReadProjects != nil {
		return filters, errReadProjects
	}
	rootRows, errRoots := s.db.Query(`SELECT DISTINCT root_workspace_name FROM ` + effectiveUsageStepsQuery() + ` WHERE root_workspace_name <> '' ORDER BY root_workspace_name`)
	if errRoots != nil {
		return filters, fmt.Errorf("querying usage root filters: %w", errRoots)
	}
	roots, errReadRoots := readStringRows(rootRows)
	if errReadRoots != nil {
		return filters, errReadRoots
	}
	filters.Projects = projects
	filters.RootProjects = roots
	if filters.Projects == nil {
		filters.Projects = []string{}
	}
	if filters.RootProjects == nil {
		filters.RootProjects = []string{}
	}
	return filters, nil
}

func readStringRows(rows *sql.Rows) ([]string, error) {
	defer func() {
		if errClose := rows.Close(); errClose != nil {
			log.Println("failed to close usage filter rows:", errClose)
		}
	}()
	var values []string
	for rows.Next() {
		var value string
		if errScan := rows.Scan(&value); errScan != nil {
			return nil, fmt.Errorf("scanning usage filter: %w", errScan)
		}
		values = append(values, value)
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, fmt.Errorf("reading usage filters: %w", errRows)
	}
	return values, nil
}

func emptyUsageResponse() usageResponse {
	return usageResponse{Daily: []usageDailyPoint{}, Rows: []usageRow{}, Filters: usageFilters{Projects: []string{}, RootProjects: []string{}}}
}

func parseUsageQuery(r *http.Request) (usageQuery, error) {
	now := time.Now().UTC()
	q := usageQuery{From: now.AddDate(0, 0, -30), To: now, Project: strings.TrimSpace(r.URL.Query().Get("project")), RootProject: strings.TrimSpace(r.URL.Query().Get("rootProject"))}
	if value := strings.TrimSpace(r.URL.Query().Get("from")); value != "" {
		parsed, errParse := time.Parse(time.DateOnly, value)
		if errParse != nil {
			return q, fmt.Errorf("invalid from date")
		}
		q.From = parsed
	}
	if value := strings.TrimSpace(r.URL.Query().Get("to")); value != "" {
		parsed, errParse := time.Parse(time.DateOnly, value)
		if errParse != nil {
			return q, fmt.Errorf("invalid to date")
		}
		q.To = parsed
	}
	q.From = dateOnly(q.From)
	q.To = dateOnly(q.To)
	if q.From.After(q.To) {
		return q, fmt.Errorf("from date must be before to date")
	}
	if strings.ContainsAny(q.Project, "/\\\x00") || strings.ContainsAny(q.RootProject, "/\\\x00") {
		return q, fmt.Errorf("invalid filter value")
	}
	return q, nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func normalizeStepTimestamp(value string) string {
	if value == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	if parsed, errParse := time.Parse(time.RFC3339, value); errParse == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func usageContextForWorkspace(workspacePath string) usageWorkspaceContext {
	ctx := usageWorkspaceContext{WorkspacePath: workspacePath, WorkspaceName: filepath.Base(workspacePath), RootWorkspacePath: workspacePath, RootWorkspaceName: filepath.Base(workspacePath)}
	if classifyWorkspace(workspacePath) == workspaceFork {
		if rootPath := getRootWorkspacePath(workspacePath); rootPath != "" {
			ctx.RootWorkspacePath = rootPath
			ctx.RootWorkspaceName = filepath.Base(rootPath)
		}
	}
	return ctx
}

func writeGlobalUsage(source, workspacePath string, sessions []state.SessionUsage) error {
	store, errStore := openGlobalUsageStore()
	if errStore != nil {
		return errStore
	}
	defer func() {
		if errClose := store.close(); errClose != nil {
			log.Println("failed to close usage database:", errClose)
		}
	}()
	return store.replaceSessionUsage(source, usageContextForWorkspace(workspacePath), sessions)
}

func (s *Server) ensureUsageStore() (*usageStore, error) {
	s.usageStoreMu.Lock()
	defer s.usageStoreMu.Unlock()
	if s.usageStore != nil || s.usageStoreErr != nil {
		return s.usageStore, s.usageStoreErr
	}
	store, errStore := openGlobalUsageStore()
	if errStore != nil {
		s.usageStoreErr = errStore
		return nil, errStore
	}
	s.usageStore = store
	return store, nil
}

func (s *Server) handleAPIUsage(w http.ResponseWriter, r *http.Request) {
	q, errQuery := parseUsageQuery(r)
	if errQuery != nil {
		http.Error(w, errQuery.Error(), http.StatusBadRequest)
		return
	}
	store, errStore := s.ensureUsageStore()
	if errStore != nil || store == nil {
		resp := emptyUsageResponse()
		resp.Warning = "usage storage unavailable"
		writeJSON(w, resp)
		return
	}
	resp, errUsage := store.query(q)
	if errUsage != nil {
		log.Println("failed to query usage:", errUsage)
		resp := emptyUsageResponse()
		resp.Warning = "usage storage unavailable"
		writeJSON(w, resp)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleAPIUsageRefresh(w http.ResponseWriter, r *http.Request) {
	s.backfillGlobalUsage()
	s.handleAPIUsage(w, r)
}

func (s *Server) backfillGlobalUsage() {
	store, errStore := s.ensureUsageStore()
	if errStore != nil || store == nil {
		log.Println("usage backfill skipped:", errStore)
		return
	}
	workspacePaths, errWorkspacePaths := s.usageBackfillWorkspacePaths()
	if errWorkspacePaths != nil {
		log.Println("usage backfill skipped:", errWorkspacePaths)
		return
	}
	for _, workspacePath := range workspacePaths {
		s.backfillWorkspaceUsage(store, workspacePath)
	}
}

func (s *Server) backfillWorkspaceUsage(store *usageStore, workspacePath string) {
	wf, ok := readWorkflowForUsageBackfill(filepath.Join(workspacePath, ".sgai", "state.json"))
	if !ok {
		return
	}
	ctx := usageContextForWorkspace(workspacePath)
	sessions := usageBackfillSessions(ctx, wf)
	if len(sessions) == 0 {
		return
	}
	if errReplace := store.replaceWorkspaceUsage(ctx, sessions); errReplace != nil {
		log.Println("usage backfill failed for", workspacePath, ":", errReplace)
	}
}

func (s *Server) backfillWorkspaceUsageBeforeRemoval(workspacePath string) {
	store, errStore := s.ensureUsageStore()
	if errStore != nil || store == nil {
		log.Println("usage backfill before removal skipped:", errStore)
		return
	}
	s.backfillWorkspaceUsage(store, workspacePath)
}

func (s *Server) usageBackfillWorkspacePaths() ([]string, error) {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return nil, errScan
	}
	seen := map[string]bool{}
	var workspacePaths []string
	addPath := func(path string) {
		if path == "" {
			return
		}
		resolved := resolveSymlinks(path)
		if resolved == "" {
			resolved = path
		}
		if seen[resolved] {
			return
		}
		seen[resolved] = true
		workspacePaths = append(workspacePaths, path)
	}
	addPath(s.rootDir)
	for _, group := range groups {
		addPath(group.Root.Directory)
		for _, fork := range group.Forks {
			addPath(fork.Directory)
		}
	}
	return workspacePaths, nil
}

func usageBackfillSessions(ctx usageWorkspaceContext, wf state.Workflow) []state.SessionUsage {
	if hasSessionSteps(wf.Cost.BySession) {
		return wf.Cost.BySession
	}
	if hasSessionSummaries(wf.Cost.BySession) {
		return wf.Cost.BySession
	}
	return agentStepSessions(ctx, wf.Cost.ByAgent)
}

func hasSessionSteps(sessions []state.SessionUsage) bool {
	for _, session := range sessions {
		if len(session.Steps) > 0 {
			return true
		}
	}
	return false
}

func hasSessionSummaries(sessions []state.SessionUsage) bool {
	for _, session := range sessions {
		if hasSessionUsage(session) {
			return true
		}
	}
	return false
}

func agentStepSessions(ctx usageWorkspaceContext, agents []state.AgentCost) []state.SessionUsage {
	var sessions []state.SessionUsage
	for index, agent := range agents {
		if len(agent.Steps) == 0 && tokenUsageTotal(agent.Tokens) == 0 && agent.Cost == 0 && agent.MeteredReportedCost == 0 && agent.APIEquivalentCost == 0 {
			continue
		}
		sessionID := agentBackfillSessionID(ctx, agent.Agent, index)
		steps := agent.Steps
		if len(steps) == 0 {
			steps = []state.StepCost{{
				StepID:                     sessionID + "-summary",
				Agent:                      agent.Agent,
				SessionID:                  sessionID,
				Cost:                       agent.Cost,
				MeteredReportedCost:        agent.MeteredReportedCost,
				APIEquivalentCost:          agent.APIEquivalentCost,
				APIEquivalentCostAvailable: agent.APIEquivalentCostAvailable,
				Tokens:                     agent.Tokens,
			}}
		} else {
			steps = make([]state.StepCost, 0, len(agent.Steps))
			for _, step := range agent.Steps {
				if step.Agent == "" {
					step.Agent = agent.Agent
				}
				if step.SessionID == "" {
					step.SessionID = sessionID
				}
				steps = append(steps, step)
			}
		}
		sessions = append(sessions, state.SessionUsage{SessionID: sessionID, Agent: agent.Agent, Model: "", Tokens: agent.Tokens, MeteredReportedCost: agent.MeteredReportedCost, APIEquivalentCost: agent.APIEquivalentCost, APIEquivalentCostAvailable: agent.APIEquivalentCostAvailable, Steps: steps})
	}
	return sessions
}

func agentBackfillSessionID(ctx usageWorkspaceContext, agent string, index int) string {
	workspace := ctx.WorkspaceName
	if workspace == "" {
		workspace = fmt.Sprintf("workspace-%d", index+1)
	}
	if agent == "" {
		return fmt.Sprintf("agent-backfill-%s-%d", workspace, index+1)
	}
	return "agent-backfill-" + workspace + "-" + agent
}

func readWorkflowForUsageBackfill(path string) (state.Workflow, bool) {
	info, errStat := os.Stat(path)
	if errStat != nil || info.Size() > usageStateMaxBytes {
		return state.Workflow{}, false
	}
	data, errRead := os.ReadFile(path)
	if errRead != nil {
		return state.Workflow{}, false
	}
	var wf state.Workflow
	if errJSON := json.Unmarshal(data, &wf); errJSON != nil {
		return state.Workflow{}, false
	}
	return wf, true
}

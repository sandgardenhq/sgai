package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

const modelsDevURL = "https://models.dev/api.json"

const modelsDevCacheTTL = 24 * time.Hour

var exportSessionBytes = func(dir, sessionID string) ([]byte, error) {
	tmpFile, errCreate := os.CreateTemp("", "sgai-opencode-export-*.json")
	if errCreate != nil {
		return nil, errCreate
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	cmd := exec.Command("opencode", "export", sessionID)
	cmd.Dir = dir
	cmd.Env = buildBaseOpenCodeEnv(dir)
	cmd.Stdout = tmpFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if errRun := cmd.Run(); errRun != nil {
		_ = tmpFile.Close()
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("opencode export failed: %w: %s", errRun, msg)
		}
		return nil, fmt.Errorf("opencode export failed: %w", errRun)
	}
	if errClose := tmpFile.Close(); errClose != nil {
		return nil, errClose
	}
	return os.ReadFile(tmpPath)
}

var fetchModelsDevCatalog = func() ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(modelsDevURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models.dev returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

type exportedStep struct {
	SessionID string
	Part      part
	Timestamp int64
	Model     string
}

type exportedSessionUsage struct {
	SessionID       string
	ParentSessionID string
	Agent           string
	Model           string
	ChildSessionIDs []string
	Steps           []exportedStep
}

type exportedTranscript struct {
	Messages []exportedMessage `json:"messages"`
}

type exportedMessage struct {
	Info  exportedMessageInfo `json:"info"`
	Parts []exportedPart      `json:"parts"`
}

type exportedMessageInfo struct {
	Metadata exportedMessageMetadata `json:"metadata"`
}

type exportedMessageMetadata struct {
	Time exportedMessageTime `json:"time"`
}

type exportedMessageTime struct {
	Created   int64 `json:"created"`
	Completed int64 `json:"completed"`
}

type exportedPart struct {
	SessionID string             `json:"sessionID"`
	Type      string             `json:"type"`
	Tool      string             `json:"tool"`
	Model     string             `json:"model"`
	Cost      float64            `json:"cost"`
	Tokens    partTokens         `json:"tokens"`
	State     *exportedToolState `json:"state"`
}

type exportedToolState struct {
	Output   toolOutput           `json:"output"`
	Metadata exportedToolMetadata `json:"metadata"`
}

type exportedToolMetadata struct {
	SessionID string `json:"sessionID"`
}

type pricingCatalog map[string]struct {
	Models map[string]struct {
		Cost map[string]any `json:"cost"`
	} `json:"models"`
}

type modelsDevCache struct {
	FetchedAt time.Time       `json:"fetchedAt"`
	Catalog   json.RawMessage `json:"catalog"`
}

type priceResult struct {
	Cost        float64
	Available   bool
	Unavailable string
}

func reconcileAgentUsage(dir string, coord *state.Coordinator, agent, sessionID, modelSpec string) error {
	if coord == nil || sessionID == "" {
		return nil
	}

	catalog, catalogErr := loadModelsDevPricingCatalog(dir, time.Now())
	visited := map[string]bool{}
	usage, err := collectExportedSessionUsage(dir, agent, sessionID, "", modelSpec, visited)
	if err != nil {
		return err
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		replaceReconciledSessions(wf, usage, catalog, catalogErr)
	}); errUpdate != nil {
		return errUpdate
	}

	if errGlobal := writeGlobalUsage("session", dir, reconciledStateSessions(coord.State(), usage)); errGlobal != nil {
		return fmt.Errorf("writing global usage: %w", errGlobal)
	}
	return nil
}

func reconciledStateSessions(wf state.Workflow, usage []exportedSessionUsage) []state.SessionUsage {
	reconciled := map[string]bool{}
	for _, session := range usage {
		reconciled[session.SessionID] = true
	}
	var sessions []state.SessionUsage
	for _, session := range wf.Cost.BySession {
		if reconciled[session.SessionID] {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func collectExportedSessionUsage(dir, agent, sessionID, parentSessionID, fallbackModel string, visited map[string]bool) ([]exportedSessionUsage, error) {
	if visited[sessionID] {
		return nil, nil
	}
	visited[sessionID] = true

	data, err := exportSessionBytes(dir, sessionID)
	if err != nil {
		return nil, err
	}
	steps, childSessionIDs, err := parseExportedSession(data, sessionID, fallbackModel)
	if err != nil {
		return nil, err
	}

	usage := exportedSessionUsage{
		SessionID:       sessionID,
		ParentSessionID: parentSessionID,
		Agent:           agent,
		Model:           fallbackModel,
		ChildSessionIDs: childSessionIDs,
		Steps:           steps,
	}
	for _, step := range steps {
		if step.Model != "" {
			usage.Model = step.Model
			break
		}
	}

	result := []exportedSessionUsage{usage}
	for _, childSessionID := range childSessionIDs {
		children, err := collectExportedSessionUsage(dir, agent, childSessionID, sessionID, fallbackModel, visited)
		if err != nil {
			return nil, err
		}
		result = append(result, children...)
	}
	return result, nil
}

func parseExportedSession(data []byte, defaultSessionID, fallbackModel string) ([]exportedStep, []string, error) {
	var transcript exportedTranscript
	if err := json.Unmarshal(bytes.TrimSpace(data), &transcript); err != nil {
		return nil, nil, err
	}

	var steps []exportedStep
	childSeen := map[string]bool{}
	for _, message := range transcript.Messages {
		timestamp := message.Info.Metadata.Time.Completed
		if timestamp == 0 {
			timestamp = message.Info.Metadata.Time.Created
		}
		for _, exportedPart := range message.Parts {
			if exportedPart.Type == "step-finish" {
				if step, ok := exportedStepFromPart(exportedPart, defaultSessionID, fallbackModel, timestamp); ok {
					steps = append(steps, step)
				}
			}
			collectExportedTaskChildSessionIDs(exportedPart, defaultSessionID, childSeen)
		}
	}

	childSessionIDs := make([]string, 0, len(childSeen))
	for childSessionID := range childSeen {
		childSessionIDs = append(childSessionIDs, childSessionID)
	}
	slices.Sort(childSessionIDs)
	return steps, childSessionIDs, nil
}

func exportedStepFromPart(exportedPart exportedPart, defaultSessionID, fallbackModel string, timestamp int64) (exportedStep, bool) {
	if exportedPart.Cost == 0 && exportedPart.Tokens.Input == 0 && exportedPart.Tokens.Output == 0 && exportedPart.Tokens.Reasoning == 0 && exportedPart.Tokens.Cache.Read == 0 && exportedPart.Tokens.Cache.Write == 0 {
		return exportedStep{}, false
	}
	sessionID := exportedPart.SessionID
	if sessionID == "" {
		sessionID = defaultSessionID
	}
	model := exportedPart.Model
	if model == "" {
		model = fallbackModel
	}
	return exportedStep{SessionID: sessionID, Part: part{SessionID: sessionID, Type: exportedPart.Type, Model: model, Cost: exportedPart.Cost, Tokens: exportedPart.Tokens}, Timestamp: timestamp, Model: model}, true
}

func collectExportedTaskChildSessionIDs(exportedPart exportedPart, parentSessionID string, childSeen map[string]bool) {
	if !isTaskToolName(exportedPart.Tool) || exportedPart.State == nil {
		return
	}
	if exportedPart.State.Metadata.SessionID != "" && exportedPart.State.Metadata.SessionID != parentSessionID {
		childSeen[exportedPart.State.Metadata.SessionID] = true
	}
	for _, sessionID := range exportedPart.State.Output.sessionIDs() {
		if sessionID != "" && sessionID != parentSessionID {
			childSeen[sessionID] = true
		}
	}
}

func replaceReconciledSessions(wf *state.Workflow, usage []exportedSessionUsage, catalog pricingCatalog, catalogErr error) {
	if !hasExportedUsageSteps(usage) {
		return
	}

	reconciled := map[string]bool{}
	for _, session := range usage {
		reconciled[session.SessionID] = true
	}

	keptSessions := wf.Cost.BySession[:0]
	for _, session := range wf.Cost.BySession {
		if !reconciled[session.SessionID] {
			keptSessions = append(keptSessions, session)
		}
	}
	wf.Cost.BySession = keptSessions

	for _, session := range usage {
		wf.Cost.BySession = append(wf.Cost.BySession, buildStateSessionUsage(session, catalog, catalogErr))
	}
	rebuildCostAggregates(wf)
}

func hasExportedUsageSteps(usage []exportedSessionUsage) bool {
	for _, session := range usage {
		if len(session.Steps) > 0 {
			return true
		}
	}
	return false
}

func buildStateSessionUsage(session exportedSessionUsage, catalog pricingCatalog, catalogErr error) state.SessionUsage {
	stateSession := state.SessionUsage{
		SessionID:       session.SessionID,
		ParentSessionID: session.ParentSessionID,
		Agent:           session.Agent,
		Model:           session.Model,
		ChildSessionIDs: session.ChildSessionIDs,
	}
	for index, step := range session.Steps {
		stepCost := buildStateStepCost(session, step, index, catalog, catalogErr)
		stateSession.Tokens.Add(stepCost.Tokens)
		stateSession.MeteredReportedCost += stepCost.MeteredReportedCost
		if stepCost.APIEquivalentCostAvailable {
			stateSession.APIEquivalentCostAvailable = true
			stateSession.APIEquivalentCost += stepCost.APIEquivalentCost
		} else if stateSession.APIEquivalentCostUnavailable == "" {
			stateSession.APIEquivalentCostUnavailable = stepCost.APIEquivalentCostUnavailable
		}
		stateSession.Steps = append(stateSession.Steps, stepCost)
	}
	return stateSession
}

func buildStateStepCost(session exportedSessionUsage, step exportedStep, index int, catalog pricingCatalog, catalogErr error) state.StepCost {
	tokens := tokenUsageFromPart(step.Part)
	pricing := priceTokens(catalog, step.Model, tokens, catalogErr)
	reportedCost := step.Part.Cost
	cost := reportedCost
	if pricing.Available {
		cost = pricing.Cost
	}
	return state.StepCost{
		StepID:                       fmt.Sprintf("%s-%s-step-%d", session.Agent, session.SessionID, index+1),
		Agent:                        session.Agent,
		SessionID:                    session.SessionID,
		Cost:                         cost,
		MeteredReportedCost:          reportedCost,
		APIEquivalentCost:            pricing.Cost,
		APIEquivalentCostAvailable:   pricing.Available,
		APIEquivalentCostUnavailable: pricing.Unavailable,
		Tokens:                       tokens,
		Timestamp:                    formatStepTimestamp(step.Timestamp),
	}
}

func tokenUsageFromPart(p part) state.TokenUsage {
	return state.TokenUsage{
		Input:      p.Tokens.Input,
		Output:     p.Tokens.Output,
		Reasoning:  p.Tokens.Reasoning,
		CacheRead:  p.Tokens.Cache.Read,
		CacheWrite: p.Tokens.Cache.Write,
	}
}

func formatStepTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return time.Unix(0, timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339)
}

func rebuildCostAggregates(wf *state.Workflow) {
	wf.Cost.TotalCost = 0
	wf.Cost.MeteredReportedCost = 0
	wf.Cost.APIEquivalentCost = 0
	wf.Cost.APIEquivalentCostAvailable = false
	wf.Cost.APIEquivalentCostUnavailable = ""
	wf.Cost.TotalTokens = state.TokenUsage{}
	wf.Cost.ByAgent = nil

	for _, session := range wf.Cost.BySession {
		wf.Cost.TotalTokens.Add(session.Tokens)
		wf.Cost.MeteredReportedCost += session.MeteredReportedCost
		if session.APIEquivalentCostAvailable {
			wf.Cost.APIEquivalentCostAvailable = true
			wf.Cost.APIEquivalentCost += session.APIEquivalentCost
		} else if wf.Cost.APIEquivalentCostUnavailable == "" {
			wf.Cost.APIEquivalentCostUnavailable = session.APIEquivalentCostUnavailable
		}

		agentIndex := slices.IndexFunc(wf.Cost.ByAgent, func(agentCost state.AgentCost) bool {
			return agentCost.Agent == session.Agent
		})
		if agentIndex == -1 {
			wf.Cost.ByAgent = append(wf.Cost.ByAgent, state.AgentCost{Agent: session.Agent})
			agentIndex = len(wf.Cost.ByAgent) - 1
		}
		agentCost := &wf.Cost.ByAgent[agentIndex]
		agentCost.Tokens.Add(session.Tokens)
		agentCost.MeteredReportedCost += session.MeteredReportedCost
		if session.APIEquivalentCostAvailable {
			agentCost.APIEquivalentCostAvailable = true
			agentCost.APIEquivalentCost += session.APIEquivalentCost
		} else if agentCost.APIEquivalentCostUnavailable == "" {
			agentCost.APIEquivalentCostUnavailable = session.APIEquivalentCostUnavailable
		}
		agentCost.Steps = append(agentCost.Steps, session.Steps...)
	}

	if wf.Cost.APIEquivalentCostAvailable {
		wf.Cost.TotalCost = wf.Cost.APIEquivalentCost
	} else {
		wf.Cost.TotalCost = wf.Cost.MeteredReportedCost
	}
	for idx := range wf.Cost.ByAgent {
		if wf.Cost.ByAgent[idx].APIEquivalentCostAvailable {
			wf.Cost.ByAgent[idx].Cost = wf.Cost.ByAgent[idx].APIEquivalentCost
		} else {
			wf.Cost.ByAgent[idx].Cost = wf.Cost.ByAgent[idx].MeteredReportedCost
		}
	}
}

func loadModelsDevPricingCatalog(dir string, now time.Time) (pricingCatalog, error) {
	cachePath := filepath.Join(dir, ".sgai", "models.dev.cache.json")
	cached, cachedErr := readModelsDevCache(cachePath)
	if cachedErr == nil && now.Sub(cached.FetchedAt) < modelsDevCacheTTL {
		return parsePricingCatalog(cached.Catalog)
	}

	data, fetchErr := fetchModelsDevCatalog()
	if fetchErr == nil {
		if err := writeModelsDevCache(cachePath, now, data); err != nil {
			return nil, err
		}
		return parsePricingCatalog(data)
	}
	if cachedErr == nil {
		catalog, err := parsePricingCatalog(cached.Catalog)
		if err == nil {
			return catalog, nil
		}
	}
	return nil, fetchErr
}

func readModelsDevCache(cachePath string) (modelsDevCache, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return modelsDevCache{}, err
	}
	var cache modelsDevCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return modelsDevCache{}, err
	}
	return cache, nil
}

func writeModelsDevCache(cachePath string, fetchedAt time.Time, catalog []byte) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	cache := modelsDevCache{FetchedAt: fetchedAt, Catalog: catalog}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}

func parsePricingCatalog(data []byte) (pricingCatalog, error) {
	var catalog pricingCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

func priceTokens(catalog pricingCatalog, model string, tokens state.TokenUsage, catalogErr error) priceResult {
	if catalogErr != nil {
		return priceResult{Unavailable: "models.dev catalog unavailable"}
	}
	cost, ok := findModelCost(catalog, model)
	if !ok {
		return priceResult{Unavailable: "models.dev pricing unavailable for " + model}
	}
	rates := selectRates(cost, tokens)
	inputRate, inputOK := floatField(rates, "input")
	outputRate, outputOK := floatField(rates, "output")
	if !inputOK || !outputOK {
		return priceResult{Unavailable: "models.dev pricing missing input/output rates for " + model}
	}
	cacheReadRate, ok := floatField(rates, "cache_read")
	if !ok {
		cacheReadRate = inputRate
	}
	cacheWriteRate, ok := floatField(rates, "cache_write")
	if !ok {
		cacheWriteRate = inputRate
	}
	reasoningRate, ok := floatField(rates, "reasoning")
	if !ok {
		reasoningRate = outputRate
	}
	costValue := float64(tokens.Input)*inputRate/1_000_000 +
		float64(tokens.CacheRead)*cacheReadRate/1_000_000 +
		float64(tokens.CacheWrite)*cacheWriteRate/1_000_000 +
		float64(tokens.Output)*outputRate/1_000_000 +
		float64(tokens.Reasoning)*reasoningRate/1_000_000
	return priceResult{Cost: costValue, Available: true}
}

func findModelCost(catalog pricingCatalog, model string) (map[string]any, bool) {
	providerID, modelID, hasProvider := strings.Cut(model, "/")
	if hasProvider {
		if provider, ok := catalog[providerID]; ok {
			if modelCost, ok := provider.Models[modelID]; ok {
				return modelCost.Cost, true
			}
			if modelCost, ok := provider.Models[model]; ok {
				return modelCost.Cost, true
			}
		}
	}
	for _, provider := range catalog {
		if modelCost, ok := provider.Models[model]; ok {
			return modelCost.Cost, true
		}
	}
	return nil, false
}

func selectRates(cost map[string]any, tokens state.TokenUsage) map[string]any {
	contextTokens := tokens.Input + tokens.CacheRead + tokens.CacheWrite
	if tiers, ok := cost["tiers"].([]any); ok {
		for _, tier := range tiers {
			tierMap, ok := tier.(map[string]any)
			if !ok {
				continue
			}
			threshold := tierThreshold(tierMap)
			if threshold > 0 && contextTokens > threshold {
				return tierMap
			}
		}
	}
	if contextTokens > 200000 {
		if over, ok := cost["context_over_200k"].(map[string]any); ok {
			return over
		}
	}
	return cost
}

func tierThreshold(tier map[string]any) int {
	tierMeta, ok := tier["tier"].(map[string]any)
	if !ok {
		return 0
	}
	value, ok := floatField(tierMeta, "size")
	if !ok {
		return 0
	}
	return int(value)
}

func floatField(values map[string]any, key string) (float64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

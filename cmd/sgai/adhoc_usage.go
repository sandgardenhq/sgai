package main

import (
	"encoding/json"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func (s *Server) reconcileAdhocUsage(workspacePath, output, modelSpec string) {
	sessionID := findAdhocSessionID([]byte(output))
	if sessionID == "" {
		message := "[usage tracking unavailable: opencode run JSON did not include a session id]"
		st := s.getAdhocState(workspacePath)
		st.mu.Lock()
		st.output.WriteString("\n" + message + "\n")
		st.mu.Unlock()
		log.Println("ad-hoc usage tracking unavailable for", workspacePath+":", "opencode run JSON did not include a session id")
		return
	}
	usage, errUsage := collectExportedSessionUsage(workspacePath, "adhoc", sessionID, "", modelSpec, map[string]bool{})
	if errUsage != nil {
		log.Println("failed to collect ad-hoc usage:", errUsage)
		return
	}
	catalog, errCatalog := loadModelsDevPricingCatalog(workspacePath, nowUTC())
	var sessions []state.SessionUsage
	for _, exported := range usage {
		sessions = append(sessions, buildStateSessionUsage(exported, catalog, errCatalog))
	}
	if errGlobal := writeGlobalUsage("adhoc", workspacePath, sessions); errGlobal != nil {
		log.Println("failed to write ad-hoc global usage:", errGlobal)
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func findAdhocSessionID(data []byte) string {
	values, errValues := parseJSONValues(data)
	if errValues != nil {
		for _, line := range strings.Split(string(data), "\n") {
			valuesFromLine, errLine := parseJSONValues([]byte(line))
			if errLine == nil {
				values = append(values, valuesFromLine...)
			}
		}
	}
	seen := map[string]bool{}
	for _, value := range values {
		if id := findSessionIDValue(value, seen); id != "" {
			return id
		}
	}
	return ""
}

func findSessionIDValue(value any, seen map[string]bool) string {
	return findSessionIDValueIn(value, seen, false)
}

func findSessionIDValueIn(value any, seen map[string]bool, sessionContext bool) string {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if id := findSessionIDValueIn(item, seen, sessionContext); id != "" {
				return id
			}
		}
	case map[string]any:
		if id := directSessionIDFromMap(typed, seen, sessionContext); id != "" {
			return id
		}
		for _, key := range sortedMapKeys(typed) {
			isSessionKey := isAdhocSessionKey(key)
			if id := findSessionIDValueIn(typed[key], seen, sessionContext || isSessionKey); id != "" {
				return id
			}
		}
	case string:
		var nested any
		if strings.Contains(typed, "session") && json.Unmarshal([]byte(typed), &nested) == nil {
			return findSessionIDValueIn(nested, seen, sessionContext)
		}
	}
	return ""
}

func directSessionIDFromMap(values map[string]any, seen map[string]bool, sessionContext bool) string {
	if sessionContext {
		if id := directStringFromKeyKinds(values, seen, []string{"id"}); id != "" {
			return id
		}
	}
	return directStringFromKeyKinds(values, seen, []string{"sessionid", "session_id", "session"})
}

func directStringFromKeyKinds(values map[string]any, seen map[string]bool, kinds []string) string {
	keys := sortedMapKeys(values)
	for _, kind := range kinds {
		for _, key := range keys {
			if strings.ToLower(key) != kind {
				continue
			}
			id := stringValue(values[key])
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			return id
		}
	}
	return ""
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func isAdhocSessionKey(key string) bool {
	lowerKey := strings.ToLower(key)
	return lowerKey == "sessionid" || lowerKey == "session_id" || lowerKey == "session"
}

package main

import (
	"log"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func (s *Server) reconcileAdhocUsage(workspacePath, sessionID, modelSpec string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		log.Println("ad-hoc usage tracking unavailable for", workspacePath+":", "opencode session id not captured")
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

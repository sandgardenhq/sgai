package main

import (
	"fmt"
	"strings"
	"time"
)

func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func isTruthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "true", "1", "on":
		return true
	default:
		return false
	}
}

func retrospectiveEnabled(metadata GoalMetadata) bool {
	return isTruthy(metadata.Retrospective)
}

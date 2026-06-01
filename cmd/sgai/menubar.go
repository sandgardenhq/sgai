package main

import (
	"net/url"
	"sync"
)

type menuBarItem struct {
	name        string
	description string
	needsInput  bool
	running     bool
	stopped     bool
	pinned      bool
}

type menuBarState struct {
	mu      sync.Mutex
	tags    map[int]menuBarAction
	nextTag int
}

type menuBarAction struct {
	actionURL string
}

func countAttention(items []menuBarItem) int {
	count := 0
	for _, item := range items {
		if item.needsInput || item.stopped {
			count++
		}
	}
	return count
}

func countRunning(items []menuBarItem) int {
	count := 0
	for _, item := range items {
		if item.running {
			count++
		}
	}
	return count
}

func countActive(items []menuBarItem) int {
	count := 0
	for _, item := range items {
		if item.running || item.stopped || item.needsInput || item.pinned {
			count++
		}
	}
	return count
}

func filterVisibleItems(items []menuBarItem) []menuBarItem {
	var result []menuBarItem
	for _, item := range items {
		if item.needsInput || item.stopped || item.pinned {
			result = append(result, item)
		}
	}
	return result
}

func formatMenuItemLabel(item menuBarItem) string {
	label := item.description
	if label == "" {
		label = item.name
	}
	switch {
	case item.needsInput:
		return "\u26A0 " + label + " (Needs Input)"
	case item.running && item.pinned:
		return "\u25B6 " + label + " (Running)"
	case item.pinned:
		return "\u25CB " + label
	case item.stopped:
		return "\u25A0 " + label + " (Stopped)"
	default:
		return label
	}
}

func workspaceItemSubpath(item menuBarItem) string {
	if item.needsInput {
		return "respond"
	}
	return "progress"
}

func workspaceURL(baseURL, name, subpath string) string {
	u, errParse := url.Parse(baseURL)
	if errParse != nil {
		return baseURL + "/workspaces/" + name + "/" + subpath
	}
	u.Path = "/workspaces/" + name + "/" + subpath
	return u.String()
}

func allocTag(state *menuBarState, action menuBarAction) int {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.nextTag++
	tag := state.nextTag
	state.tags[tag] = action
	return tag
}

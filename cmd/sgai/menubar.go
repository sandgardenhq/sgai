package main

import (
	"net/url"
	"sync"
)

type menuBarItem struct {
	name       string
	needsInput bool
	running    bool
	stopped    bool
}

type menuBarState struct {
	mu      sync.Mutex
	tags    map[int]menuBarAction
	nextTag int
}

type menuBarAction struct {
	actionURL string
}

var globalMenuBar = &menuBarState{
	tags: make(map[int]menuBarAction),
}

func toMenuBarItem(w workspaceInfo) menuBarItem {
	return menuBarItem{
		name:       w.DirName,
		needsInput: w.NeedsInput,
		running:    w.Running,
		stopped:    !w.Running && w.InProgress,
	}
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
		if item.running || item.stopped || item.needsInput {
			count++
		}
	}
	return count
}

func filterAttentionItems(items []menuBarItem) []menuBarItem {
	var result []menuBarItem
	for _, item := range items {
		if item.needsInput || item.stopped {
			result = append(result, item)
		}
	}
	return result
}

func formatMenuItemLabel(item menuBarItem) string {
	if item.needsInput {
		return "\u26A0 " + item.name + " (Needs Input)"
	}
	return "\u25A0 " + item.name + " (Stopped)"
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

func allocTag(action menuBarAction) int {
	globalMenuBar.mu.Lock()
	defer globalMenuBar.mu.Unlock()
	globalMenuBar.nextTag++
	tag := globalMenuBar.nextTag
	globalMenuBar.tags[tag] = action
	return tag
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountAttention(t *testing.T) {
	tests := []struct {
		name     string
		items    []menuBarItem
		expected int
	}{
		{
			name:     "emptyItems",
			items:    []menuBarItem{},
			expected: 0,
		},
		{
			name: "needsInput",
			items: []menuBarItem{
				{needsInput: true},
			},
			expected: 1,
		},
		{
			name: "stopped",
			items: []menuBarItem{
				{stopped: true},
			},
			expected: 1,
		},
		{
			name: "running",
			items: []menuBarItem{
				{running: true},
			},
			expected: 0,
		},
		{
			name: "pinned",
			items: []menuBarItem{
				{pinned: true},
			},
			expected: 0,
		},
		{
			name: "mixedItems",
			items: []menuBarItem{
				{needsInput: true},
				{running: true},
				{stopped: true},
				{pinned: true},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countAttention(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountRunning(t *testing.T) {
	tests := []struct {
		name     string
		items    []menuBarItem
		expected int
	}{
		{
			name:     "emptyItems",
			items:    []menuBarItem{},
			expected: 0,
		},
		{
			name: "running",
			items: []menuBarItem{
				{running: true},
			},
			expected: 1,
		},
		{
			name: "notRunning",
			items: []menuBarItem{
				{running: false},
			},
			expected: 0,
		},
		{
			name: "mixedItems",
			items: []menuBarItem{
				{running: true},
				{running: false},
				{running: true},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countRunning(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountActive(t *testing.T) {
	tests := []struct {
		name     string
		items    []menuBarItem
		expected int
	}{
		{
			name:     "emptyItems",
			items:    []menuBarItem{},
			expected: 0,
		},
		{
			name: "running",
			items: []menuBarItem{
				{running: true},
			},
			expected: 1,
		},
		{
			name: "stopped",
			items: []menuBarItem{
				{stopped: true},
			},
			expected: 1,
		},
		{
			name: "needsInput",
			items: []menuBarItem{
				{needsInput: true},
			},
			expected: 1,
		},
		{
			name: "pinned",
			items: []menuBarItem{
				{pinned: true},
			},
			expected: 1,
		},
		{
			name: "inactive",
			items: []menuBarItem{
				{running: false, stopped: false, needsInput: false, pinned: false},
			},
			expected: 0,
		},
		{
			name: "mixedItems",
			items: []menuBarItem{
				{running: true},
				{stopped: true},
				{needsInput: true},
				{pinned: true},
				{running: false, stopped: false, needsInput: false, pinned: false},
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countActive(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterVisibleItems(t *testing.T) {
	tests := []struct {
		name     string
		items    []menuBarItem
		expected int
	}{
		{
			name:     "emptyItems",
			items:    []menuBarItem{},
			expected: 0,
		},
		{
			name: "needsInput",
			items: []menuBarItem{
				{needsInput: true, name: "test"},
			},
			expected: 1,
		},
		{
			name: "stopped",
			items: []menuBarItem{
				{stopped: true, name: "test"},
			},
			expected: 1,
		},
		{
			name: "pinned",
			items: []menuBarItem{
				{pinned: true, name: "test"},
			},
			expected: 1,
		},
		{
			name: "runningNotVisible",
			items: []menuBarItem{
				{running: true, name: "test"},
			},
			expected: 0,
		},
		{
			name: "mixedItems",
			items: []menuBarItem{
				{needsInput: true, name: "a"},
				{running: true, name: "b"},
				{stopped: true, name: "c"},
				{pinned: true, name: "d"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterVisibleItems(tt.items)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestFormatMenuItemLabel(t *testing.T) {
	tests := []struct {
		name     string
		item     menuBarItem
		expected string
	}{
		{
			name: "needsInput",
			item: menuBarItem{
				name:        "workspace",
				description: "Test Goal",
				needsInput:  true,
			},
			expected: "\u26A0 Test Goal (Needs Input)",
		},
		{
			name: "runningAndPinned",
			item: menuBarItem{
				name:        "workspace",
				description: "Test Goal",
				running:     true,
				pinned:      true,
			},
			expected: "\u25B6 Test Goal (Running)",
		},
		{
			name: "pinnedOnly",
			item: menuBarItem{
				name:        "workspace",
				description: "Test Goal",
				pinned:      true,
			},
			expected: "\u25CB Test Goal",
		},
		{
			name: "stopped",
			item: menuBarItem{
				name:        "workspace",
				description: "Test Goal",
				stopped:     true,
			},
			expected: "\u25A0 Test Goal (Stopped)",
		},
		{
			name: "default",
			item: menuBarItem{
				name:        "workspace",
				description: "Test Goal",
			},
			expected: "Test Goal",
		},
		{
			name: "noDescription",
			item: menuBarItem{
				name: "workspace",
			},
			expected: "workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMenuItemLabel(tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkspaceItemSubpath(t *testing.T) {
	tests := []struct {
		name     string
		item     menuBarItem
		expected string
	}{
		{
			name:     "needsInput",
			item:     menuBarItem{needsInput: true},
			expected: "respond",
		},
		{
			name:     "notNeedsInput",
			item:     menuBarItem{needsInput: false},
			expected: "progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workspaceItemSubpath(tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkspaceURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		workspace string
		subpath   string
		expected  string
	}{
		{
			name:      "validURL",
			baseURL:   "http://localhost:8080",
			workspace: "my-workspace",
			subpath:   "progress",
			expected:  "http://localhost:8080/workspaces/my-workspace/progress",
		},
		{
			name:      "urlWithTrailingSlash",
			baseURL:   "http://localhost:8080/",
			workspace: "my-workspace",
			subpath:   "progress",
			expected:  "http://localhost:8080/workspaces/my-workspace/progress",
		},
		{
			name:      "urlWithPath",
			baseURL:   "http://localhost:8080/some/path",
			workspace: "my-workspace",
			subpath:   "progress",
			expected:  "http://localhost:8080/workspaces/my-workspace/progress",
		},
		{
			name:      "invalidURL",
			baseURL:   "://invalid",
			workspace: "my-workspace",
			subpath:   "progress",
			expected:  "://invalid/workspaces/my-workspace/progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workspaceURL(tt.baseURL, tt.workspace, tt.subpath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllocTag(t *testing.T) {
	state := &menuBarState{
		tags:    make(map[int]menuBarAction),
		nextTag: 0,
	}

	tag1 := allocTag(state, menuBarAction{actionURL: "url1"})
	assert.Equal(t, 1, tag1)

	tag2 := allocTag(state, menuBarAction{actionURL: "url2"})
	assert.Equal(t, 2, tag2)

	assert.Len(t, state.tags, 2)
	assert.Equal(t, "url1", state.tags[1].actionURL)
	assert.Equal(t, "url2", state.tags[2].actionURL)
}

func TestFormatMenuItemLabelVariants(t *testing.T) {
	t.Run("needsInput", func(t *testing.T) {
		label := formatMenuItemLabel(menuBarItem{name: "ws", needsInput: true})
		assert.Contains(t, label, "Needs Input")
	})
	t.Run("stopped", func(t *testing.T) {
		label := formatMenuItemLabel(menuBarItem{name: "ws", stopped: true})
		assert.Contains(t, label, "Stopped")
	})
	t.Run("pinnedRunning", func(t *testing.T) {
		label := formatMenuItemLabel(menuBarItem{name: "ws", running: true, pinned: true})
		assert.Contains(t, label, "Running")
	})
	t.Run("idle", func(t *testing.T) {
		label := formatMenuItemLabel(menuBarItem{name: "ws"})
		assert.Equal(t, "ws", label)
	})
}

func TestFilterVisibleItemsResult(t *testing.T) {
	items := []menuBarItem{
		{name: "running-ws", running: true},
		{name: "idle-ws"},
		{name: "pinned-ws", pinned: true},
		{name: "input-ws", needsInput: true},
		{name: "stopped-ws", stopped: true},
	}
	filtered := filterVisibleItems(items)
	assert.Len(t, filtered, 3)
}

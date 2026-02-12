package main

import (
	"testing"
)

func TestCountAttention(t *testing.T) {
	cases := []struct {
		name  string
		items []menuBarItem
		want  int
	}{
		{
			name:  "empty",
			items: nil,
			want:  0,
		},
		{
			name: "noAttention",
			items: []menuBarItem{
				{name: "a", running: true},
				{name: "b", running: true},
			},
			want: 0,
		},
		{
			name: "needsInput",
			items: []menuBarItem{
				{name: "a", needsInput: true},
				{name: "b", running: true},
			},
			want: 1,
		},
		{
			name: "stopped",
			items: []menuBarItem{
				{name: "a", stopped: true},
				{name: "b", running: true},
			},
			want: 1,
		},
		{
			name: "multipleAttention",
			items: []menuBarItem{
				{name: "a", needsInput: true},
				{name: "b", stopped: true},
				{name: "c", running: true},
			},
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countAttention(tc.items)
			if got != tc.want {
				t.Errorf("countAttention() = %d; want %d", got, tc.want)
			}
		})
	}
}

func TestCountRunning(t *testing.T) {
	cases := []struct {
		name  string
		items []menuBarItem
		want  int
	}{
		{
			name:  "empty",
			items: nil,
			want:  0,
		},
		{
			name: "twoRunning",
			items: []menuBarItem{
				{name: "a", running: true},
				{name: "b", running: true},
				{name: "c"},
			},
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countRunning(tc.items)
			if got != tc.want {
				t.Errorf("countRunning() = %d; want %d", got, tc.want)
			}
		})
	}
}

func TestCountActive(t *testing.T) {
	cases := []struct {
		name  string
		items []menuBarItem
		want  int
	}{
		{
			name:  "empty",
			items: nil,
			want:  0,
		},
		{
			name: "allRunning",
			items: []menuBarItem{
				{name: "a", running: true},
				{name: "b", running: true},
			},
			want: 2,
		},
		{
			name: "mixed",
			items: []menuBarItem{
				{name: "a", running: true},
				{name: "b", stopped: true},
				{name: "c", needsInput: true},
				{name: "d"},
			},
			want: 3,
		},
		{
			name: "noneActive",
			items: []menuBarItem{
				{name: "a"},
				{name: "b"},
			},
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countActive(tc.items)
			if got != tc.want {
				t.Errorf("countActive() = %d; want %d", got, tc.want)
			}
		})
	}
}

func TestFilterAttentionItems(t *testing.T) {
	items := []menuBarItem{
		{name: "a", running: true},
		{name: "b", needsInput: true},
		{name: "c", stopped: true},
		{name: "d", running: true},
	}

	got := filterAttentionItems(items)
	if len(got) != 2 {
		t.Fatalf("filterAttentionItems() returned %d items; want 2", len(got))
	}
	if got[0].name != "b" {
		t.Errorf("filterAttentionItems()[0].name = %q; want %q", got[0].name, "b")
	}
	if got[1].name != "c" {
		t.Errorf("filterAttentionItems()[1].name = %q; want %q", got[1].name, "c")
	}
}

func TestFilterAttentionItemsEmpty(t *testing.T) {
	items := []menuBarItem{
		{name: "a", running: true},
	}
	got := filterAttentionItems(items)
	if len(got) != 0 {
		t.Errorf("filterAttentionItems() returned %d items; want 0", len(got))
	}
}

func TestFormatMenuItemLabel(t *testing.T) {
	cases := []struct {
		name string
		item menuBarItem
		want string
	}{
		{
			name: "needsInput",
			item: menuBarItem{name: "my-workspace", needsInput: true},
			want: "\u26A0 my-workspace (Needs Input)",
		},
		{
			name: "stopped",
			item: menuBarItem{name: "my-workspace", stopped: true},
			want: "\u25A0 my-workspace (Stopped)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatMenuItemLabel(tc.item)
			if got != tc.want {
				t.Errorf("formatMenuItemLabel() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestWorkspaceURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		wsName  string
		subpath string
		want    string
	}{
		{
			name:    "respondRoute",
			baseURL: "http://127.0.0.1:8080",
			wsName:  "my-project",
			subpath: "respond",
			want:    "http://127.0.0.1:8080/workspaces/my-project/respond",
		},
		{
			name:    "progressRoute",
			baseURL: "http://127.0.0.1:8080",
			wsName:  "my-project",
			subpath: "progress",
			want:    "http://127.0.0.1:8080/workspaces/my-project/progress",
		},
		{
			name:    "customPort",
			baseURL: "http://localhost:9090",
			wsName:  "test-ws",
			subpath: "progress",
			want:    "http://localhost:9090/workspaces/test-ws/progress",
		},
		{
			name:    "invalidURL",
			baseURL: "://invalid",
			wsName:  "test",
			subpath: "respond",
			want:    "://invalid/workspaces/test/respond",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := workspaceURL(tc.baseURL, tc.wsName, tc.subpath)
			if got != tc.want {
				t.Errorf("workspaceURL(%q, %q, %q) = %q; want %q", tc.baseURL, tc.wsName, tc.subpath, got, tc.want)
			}
		})
	}
}

func TestWorkspaceItemSubpath(t *testing.T) {
	cases := []struct {
		name string
		item menuBarItem
		want string
	}{
		{
			name: "needsInput",
			item: menuBarItem{name: "ws", needsInput: true},
			want: "respond",
		},
		{
			name: "stopped",
			item: menuBarItem{name: "ws", stopped: true},
			want: "progress",
		},
		{
			name: "running",
			item: menuBarItem{name: "ws", running: true},
			want: "progress",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := workspaceItemSubpath(tc.item)
			if got != tc.want {
				t.Errorf("workspaceItemSubpath() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestAllocTag(t *testing.T) {
	saved := globalMenuBar
	t.Cleanup(func() { globalMenuBar = saved })

	globalMenuBar = &menuBarState{
		tags: make(map[int]menuBarAction),
	}

	tag1 := allocTag(menuBarAction{actionURL: "http://example.com"})
	tag2 := allocTag(menuBarAction{actionURL: "http://other.com"})

	if tag1 == tag2 {
		t.Errorf("allocTag returned duplicate tags: %d", tag1)
	}
	if tag1 < 1 {
		t.Errorf("allocTag returned non-positive tag: %d", tag1)
	}

	globalMenuBar.mu.Lock()
	action, ok := globalMenuBar.tags[tag1]
	globalMenuBar.mu.Unlock()
	if !ok {
		t.Errorf("allocTag did not store action for tag %d", tag1)
	}
	if action.actionURL != "http://example.com" {
		t.Errorf("stored action URL = %q; want %q", action.actionURL, "http://example.com")
	}
}

func TestToMenuBarItem(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		w := workspaceInfo{
			DirName:    "test-workspace",
			Running:    true,
			NeedsInput: false,
			InProgress: true,
		}
		got := toMenuBarItem(w)
		if got.name != "test-workspace" {
			t.Errorf("name = %q; want %q", got.name, "test-workspace")
		}
		if !got.running {
			t.Error("expected running = true")
		}
		if got.needsInput {
			t.Error("expected needsInput = false")
		}
		if got.stopped {
			t.Error("expected stopped = false (running overrides)")
		}
	})

	t.Run("stoppedAfterProgress", func(t *testing.T) {
		w := workspaceInfo{
			DirName:    "stopped-ws",
			Running:    false,
			NeedsInput: false,
			InProgress: true,
		}
		got := toMenuBarItem(w)
		if !got.stopped {
			t.Error("expected stopped = true (not running but in progress)")
		}
	})

	t.Run("idle", func(t *testing.T) {
		w := workspaceInfo{
			DirName:    "idle-ws",
			Running:    false,
			NeedsInput: false,
			InProgress: false,
		}
		got := toMenuBarItem(w)
		if got.stopped {
			t.Error("expected stopped = false (never started)")
		}
	})
}

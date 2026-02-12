//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

extern void MenuBarInit(void);
extern void MenuBarSetTitle(const char *title);
extern void MenuBarClear(void);
extern void MenuBarAddItem(const char *title, int tag, int enabled);
extern void MenuBarAddSeparator(void);
extern void MenuBarOpenURL(const char *urlStr);
extern void MenuBarRunLoop(void);
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

var menuBarBaseURL string

//export goMenuItemClicked
func goMenuItemClicked(tag C.int) {
	globalMenuBar.mu.Lock()
	action, ok := globalMenuBar.tags[int(tag)]
	globalMenuBar.mu.Unlock()
	if !ok || action.actionURL == "" {
		return
	}
	cURL := C.CString(action.actionURL)
	defer C.free(unsafe.Pointer(cURL))
	C.MenuBarOpenURL(cURL)
}

func startMenuBar(baseURL string, srv *Server) {
	menuBarBaseURL = baseURL

	runtime.LockOSThread()
	C.MenuBarInit()

	go menuBarUpdateLoop(srv)

	C.MenuBarRunLoop()
}

func menuBarUpdateLoop(srv *Server) {
	ch := srv.sseBroker.subscribe()
	defer srv.sseBroker.unsubscribe(ch)

	rebuildMenuFromServer(srv)

	for {
		select {
		case <-ch.done:
			return
		case <-ch.events:
			rebuildMenuFromServer(srv)
		}
	}
}

func rebuildMenuFromServer(srv *Server) {
	groups, errScan := srv.scanWorkspaceGroups()
	if errScan != nil {
		return
	}

	var items []menuBarItem
	for _, grp := range groups {
		items = append(items, toMenuBarItem(grp.Root))
		for _, fork := range grp.Forks {
			items = append(items, toMenuBarItem(fork))
		}
	}

	globalMenuBar.mu.Lock()
	globalMenuBar.nextTag = 0
	globalMenuBar.tags = make(map[int]menuBarAction)
	baseURL := menuBarBaseURL
	globalMenuBar.mu.Unlock()

	attentionCount := countAttention(items)
	setMenuTitle(attentionCount)

	C.MenuBarClear()

	dashTag := allocTag(menuBarAction{actionURL: baseURL})
	addMenuEntry("Open Dashboard", dashTag, true)

	C.MenuBarAddSeparator()

	needsAttention := filterAttentionItems(items)
	if len(needsAttention) == 0 {
		addMenuEntry("No factories need attention", 0, false)
	} else {
		summary := fmt.Sprintf("%d factory(ies) need attention", len(needsAttention))
		addMenuEntry(summary, 0, false)
		C.MenuBarAddSeparator()
		for _, item := range needsAttention {
			label := formatMenuItemLabel(item)
			itemURL := workspaceURL(baseURL, item.name, workspaceItemSubpath(item))
			tag := allocTag(menuBarAction{actionURL: itemURL})
			addMenuEntry(label, tag, true)
		}
	}

	C.MenuBarAddSeparator()

	runningCount := countRunning(items)
	statusLine := fmt.Sprintf("%d running, %d need attention", runningCount, attentionCount)
	addMenuEntry(statusLine, 0, false)
}

func setMenuTitle(attentionCount int) {
	var title string
	if attentionCount > 0 {
		title = fmt.Sprintf("\u25CF %d", attentionCount)
	} else {
		title = "\u25CB sgai"
	}
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.MenuBarSetTitle(cTitle)
}

func addMenuEntry(label string, tag int, enabled bool) {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	e := 0
	if enabled {
		e = 1
	}
	C.MenuBarAddItem(cLabel, C.int(tag), C.int(e))
}

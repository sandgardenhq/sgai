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
extern void MenuBarStop(void);
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

var menuBarBaseURL string

var menuBarCancelFunc context.CancelFunc

//export goMenuItemClicked
func goMenuItemClicked(tag C.int) {
	globalMenuBar.mu.Lock()
	action, ok := globalMenuBar.tags[int(tag)]
	globalMenuBar.mu.Unlock()
	if !ok {
		return
	}
	if action.actionURL == "" {
		if menuBarCancelFunc != nil {
			menuBarCancelFunc()
		}
		return
	}
	cURL := C.CString(action.actionURL)
	defer C.free(unsafe.Pointer(cURL))
	C.MenuBarOpenURL(cURL)
}

func startMenuBar(ctx context.Context, baseURL string, srv *Server, cancel context.CancelFunc) {
	menuBarBaseURL = baseURL
	menuBarCancelFunc = cancel

	C.MenuBarInit()

	go menuBarUpdateLoop(ctx, srv)
	go func() {
		<-ctx.Done()
		C.MenuBarStop()
	}()

	C.MenuBarRunLoop()
}

func menuBarUpdateLoop(ctx context.Context, srv *Server) {
	ch := srv.sseBroker.subscribe()
	defer srv.sseBroker.unsubscribe(ch)

	rebuildMenuFromServer(srv)

	for {
		select {
		case <-ctx.Done():
			return
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

	runningCount := countRunning(items)
	attentionCount := countAttention(items)
	totalActive := countActive(items)
	setMenuTitle(runningCount, totalActive, attentionCount)

	C.MenuBarClear()

	dashTag := allocTag(menuBarAction{actionURL: baseURL})
	addMenuEntry("Open Dashboard", dashTag, true)

	C.MenuBarAddSeparator()

	for _, item := range filterVisibleItems(items) {
		label := formatMenuItemLabel(item)
		itemURL := workspaceURL(baseURL, item.name, workspaceItemSubpath(item))
		tag := allocTag(menuBarAction{actionURL: itemURL})
		addMenuEntry(label, tag, true)
	}

	C.MenuBarAddSeparator()
	quitTag := allocTag(menuBarAction{actionURL: ""})
	addMenuEntry("Quit", quitTag, true)
}

func setMenuTitle(runningCount, totalActive, attentionCount int) {
	var title string
	switch {
	case totalActive == 0:
		title = "\u25CF sgai"
	case attentionCount > 0:
		title = fmt.Sprintf("\u26A0 %d/%d", runningCount, totalActive)
	default:
		title = fmt.Sprintf("\u25CF %d/%d", runningCount, totalActive)
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

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

type darwinMenuBarState struct {
	menuBarState
	baseURL    string
	cancelFunc context.CancelFunc
}

var menuBarClickCh = make(chan int, 1)

//export goMenuItemClicked
func goMenuItemClicked(tag C.int) {
	select {
	case menuBarClickCh <- int(tag):
	default:
	}
}

func startMenuBar(ctx context.Context, baseURL string, srv *Server, cancel context.CancelFunc) {
	state := &darwinMenuBarState{
		menuBarState: menuBarState{tags: make(map[int]menuBarAction)},
		baseURL:      baseURL,
		cancelFunc:   cancel,
	}

	C.MenuBarInit()

	go menuBarClickHandler(ctx, state)
	go menuBarUpdateLoop(ctx, srv, state)
	go func() {
		<-ctx.Done()
		C.MenuBarStop()
	}()

	C.MenuBarRunLoop()
}

func menuBarClickHandler(ctx context.Context, state *darwinMenuBarState) {
	for {
		select {
		case <-ctx.Done():
			return
		case tag := <-menuBarClickCh:
			state.mu.Lock()
			action, ok := state.tags[tag]
			cancel := state.cancelFunc
			state.mu.Unlock()
			if !ok {
				continue
			}
			if action.actionURL == "" {
				if cancel != nil {
					cancel()
				}
				continue
			}
			cURL := C.CString(action.actionURL)
			C.MenuBarOpenURL(cURL)
			C.free(unsafe.Pointer(cURL))
		}
	}
}

func menuBarUpdateLoop(ctx context.Context, srv *Server, state *darwinMenuBarState) {
	sub := srv.signals.subscribe()
	defer srv.signals.unsubscribe(sub)

	rebuildMenuFromServer(srv, state)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.done:
			return
		case <-sub.ch:
			rebuildMenuFromServer(srv, state)
		}
	}
}

func rebuildMenuFromServer(srv *Server, state *darwinMenuBarState) {
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

	state.mu.Lock()
	state.nextTag = 0
	state.tags = make(map[int]menuBarAction)
	baseURL := state.baseURL
	state.mu.Unlock()

	runningCount := countRunning(items)
	attentionCount := countAttention(items)
	totalActive := countActive(items)
	setMenuTitle(runningCount, totalActive, attentionCount)

	C.MenuBarClear()

	dashTag := allocTag(&state.menuBarState, menuBarAction{actionURL: baseURL})
	addMenuEntry("Open Dashboard", dashTag, true)

	C.MenuBarAddSeparator()

	for _, item := range filterVisibleItems(items) {
		label := formatMenuItemLabel(item)
		itemURL := workspaceURL(baseURL, item.name, workspaceItemSubpath(item))
		tag := allocTag(&state.menuBarState, menuBarAction{actionURL: itemURL})
		addMenuEntry(label, tag, true)
	}

	C.MenuBarAddSeparator()
	quitTag := allocTag(&state.menuBarState, menuBarAction{actionURL: ""})
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

//go:build !darwin

package main

import "context"

func startMenuBar(ctx context.Context, _ string, _ *Server, _ context.CancelFunc) {
	<-ctx.Done()
}

//go:build !darwin

package main

func startMenuBar(_ string, _ *Server) {
	select {}
}

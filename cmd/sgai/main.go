// Command sgai is CLI for AI-powered software factory
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

const workGateApprovalText = "DEFINITION IS COMPLETE, BUILD MAY BEGIN"

const maxConsecutiveWorkingIterations = 50

func main() {
	runtime.LockOSThread()
	subcommand := ""
	if len(os.Args) >= 2 {
		subcommand = os.Args[1]
	}

	if requiresOpencode(subcommand) {
		if _, err := exec.LookPath("opencode"); err != nil {
			log.Fatalln("opencode is required but not found in PATH")
		}
	}

	switch subcommand {
	case "internal-mcp":
		if err := runInternalMCP(context.Background(), os.Args[2:]); err != nil {
			log.Fatalln("internal mcp failed:", err)
		}
		return
	case "help", "-h", "--help":
		printUsage()
		return
	case "serve":
		cmdServe(os.Args[2:])
		return
	default:
		cmdServe(os.Args[1:])
		return
	}
}

func requiresOpencode(subcommand string) bool {
	switch subcommand {
	case "help", "-h", "--help", "internal-mcp":
		return false
	default:
		return true
	}
}

func printUsage() {
	fmt.Println(`sgai - AI-powered software factory

Usage:
  sgai [--listen-addr addr]    Start web server (default)

Options:
  --listen-addr   HTTP server listen address (default: 127.0.0.1:8080)

Examples:
  sgai
      Start web UI on localhost:8080
  sgai --listen-addr 0.0.0.0:8080
      Start web UI accessible externally`)
}

---
name: go-project-layout
description: Go module structure and project organization patterns. When setting up a new Go project or organizing existing Go code
metadata:
    languages: go
---

# Go Project Layout

Module structure and organization patterns for Go projects.

## Reference URLs

For deeper information, fetch these URLs:
- https://go.dev/doc/code - How to Write Go Code
- https://go.dev/doc/modules/layout - Go Module Layout

## Module Basics

### Initialize Module

```bash
# Create new module
mkdir myproject
cd myproject
go mod init github.com/user/myproject
```

### go.mod File

```go
module github.com/user/myproject

go 1.23  // Use Go 1.23+ for iter package and range-over-func

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/stretchr/testify v1.8.4
)
```

**Go Version Requirements:**
- `go 1.18+` - Generics support
- `go 1.21+` - `slices` and `maps` packages in stdlib
- `go 1.23+` - `iter` package and range-over-function iterators

### Managing Dependencies

```bash
go mod tidy        # Add missing, remove unused
go mod download    # Download dependencies
go mod verify      # Verify dependencies
go mod graph       # View dependency graph
go get ./...       # Update dependencies
go get pkg@version # Get specific version
```

## Standard Project Structure

### Minimal Project

```
myproject/
├── go.mod
├── go.sum
├── main.go        # Simple single-file projects
└── main_test.go
```

### Application Project

```
myproject/
├── cmd/
│   └── myapp/
│       └── main.go       # Application entry point
├── internal/
│   ├── server/
│   │   └── server.go     # HTTP server
│   ├── handler/
│   │   └── handler.go    # Request handlers
│   └── model/
│       └── model.go      # Data models
├── go.mod
├── go.sum
└── README.md
```

### Library Project

```
mylib/
├── go.mod
├── go.sum
├── mylib.go           # Main package code
├── mylib_test.go      # Tests
├── helper.go          # Additional files
└── README.md
```

### Full Application

```
myproject/
├── cmd/
│   ├── myapp/
│   │   └── main.go
│   └── mytool/
│       └── main.go
├── internal/
│   ├── pkg/           # Private shared packages
│   │   └── database/
│   └── app/
│       └── myapp/
├── pkg/               # Public packages (if any)
│   └── api/
├── api/               # API definitions (OpenAPI, proto)
├── configs/           # Configuration files
├── scripts/           # Build scripts
├── test/              # Additional test data
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Directory Conventions

### /cmd

Application entry points. Each subdirectory is a separate binary.

```go
// cmd/myapp/main.go
package main

import (
    "github.com/user/myproject/internal/server"
)

func main() {
    s := server.New()
    s.Run()
}
```

### /internal

Private packages. Cannot be imported by external projects.

```go
// internal/server/server.go
package server

type Server struct {
    // ...
}

func New() *Server {
    return &Server{}
}
```

### /pkg

Public packages. Can be imported by external projects.
**Note:** Only use if you explicitly want to share code.

### /api

API definitions: OpenAPI specs, Protocol Buffers, etc.

## Package Organization

### One Package Per Directory

Each directory is one package. All `.go` files in a directory must have the same `package` statement.

### Package Naming

- Short, lowercase, no underscores
- Name reflects purpose
- Avoid `util`, `common`, `misc`

```go
// Good
package server
package handler
package model

// Bad
package serverPkg
package server_utils
package common
```

### Import Paths

Import path = module path + subdirectory.

```go
// go.mod: module github.com/user/myproject

// Import internal package
import "github.com/user/myproject/internal/server"

// Import cmd package (rare, usually just main)
import "github.com/user/myproject/cmd/myapp"
```

## Building and Installing

### Build Commands

```bash
# Build current directory
go build

# Build specific package
go build ./cmd/myapp

# Build all packages
go build ./...

# Install binary
go install ./cmd/myapp

# Run without installing
go run ./cmd/myapp
```

### Cross-Compilation

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o myapp-linux ./cmd/myapp

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o myapp.exe ./cmd/myapp

# Build for macOS ARM
GOOS=darwin GOARCH=arm64 go build -o myapp-darwin ./cmd/myapp
```

## Common Patterns

### Multiple Binaries

```
cmd/
├── api-server/
│   └── main.go
├── worker/
│   └── main.go
└── cli-tool/
    └── main.go
```

### Shared Internal Code

```
internal/
├── config/        # Configuration loading
│   └── config.go
├── database/      # Database connection
│   └── db.go
└── model/         # Shared models
    └── user.go
```

### Service Layer

```
internal/
├── handler/       # HTTP handlers
├── service/       # Business logic
├── repository/    # Data access
└── model/         # Data structures
```

## Testing Structure

### Test Files

Tests live alongside code with `_test.go` suffix.

```
internal/
└── server/
    ├── server.go
    └── server_test.go
```

### Test Data

```
test/
├── testdata/
│   └── fixtures/
└── integration/
```

### External Test Package

```go
// server_test.go - test from outside perspective
package server_test

import (
    "testing"
    "github.com/user/myproject/internal/server"
)

func TestServer(t *testing.T) {
    s := server.New()
    // ...
}
```

## Makefile Example

```makefile
.PHONY: build test lint clean

build:
	go build -o bin/myapp ./cmd/myapp

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -rf bin/

run:
	go run ./cmd/myapp
```

## Quick Reference

| Directory | Purpose | Importable |
|-----------|---------|------------|
| /cmd | Entry points | No (main packages) |
| /internal | Private packages | Only within module |
| /pkg | Public packages | Yes |
| /api | API definitions | N/A |
| /configs | Config files | N/A |
| /test | Test helpers | N/A |

## Modern Go Features

When setting up a new project, consider which modern Go features you'll use:

### Go Version Selection

| Feature | Minimum Go Version |
|---------|-------------------|
| Generics | Go 1.18 |
| `slices` package | Go 1.21 |
| `maps` package | Go 1.21 |
| `iter` package | Go 1.23 |
| Range-over-func | Go 1.23 |

Set your `go.mod` version accordingly to leverage these features:

```go
// For full modern Go feature support
go 1.23

// For generics + slices/maps but no iterators
go 1.21
```

**References:**
- https://pkg.go.dev/slices
- https://pkg.go.dev/maps
- https://pkg.go.dev/iter
- https://go.dev/doc/tutorial/generics

## Common Mistakes

- **Too many packages** - Don't over-organize
- **Circular imports** - Restructure to break cycles
- **Everything in /pkg** - Use /internal by default
- **main in root** - Use /cmd for applications
- **Generic package names** - Be specific
- **Wrong Go version** - Use Go 1.21+ for modern stdlib features

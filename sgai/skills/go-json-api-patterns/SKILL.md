---
name: go-json-api-patterns
description: "/api/v1/* JSON endpoints in serve_api.go, shared business logic extraction, SSE event emission patterns, idempotent endpoints. Use when creating Go JSON API endpoints for the React SPA, extracting shared logic from serve.go, implementing SSE event publishing, or designing idempotent mutation endpoints. Triggers on serve_api.go, /api/v1/, JSON endpoint, SSE publish, shared business logic tasks."
---

# Go JSON API Patterns

## Overview

Guide for writing `/api/v1/*` JSON endpoints in `serve_api.go` that share business logic with existing HTMX handlers in `serve.go`. Covers endpoint structure, shared function extraction, JSON error format, SSE event emission via structural middleware, idempotent endpoint design, and content negotiation.

**STPA References:** R-4 (structural SSE emission), R-5 (post-commit publishing), R-10 (idempotent endpoints), R-20 (deferred event publishing).

## When to Use

- Use when creating new `/api/v1/*` JSON endpoints
- Use when extracting shared business logic from `serve.go` into common functions
- Use when implementing SSE event emission for state changes
- Use when designing mutation endpoints (start/stop/respond/create)
- Don't use for HTMX handler modifications (those stay in `serve.go` unchanged)

## Architecture

```
serve.go          HTMX handlers (EXISTING, UNCHANGED)
    │
    └── shared business logic ──► extracted functions
                                      │
serve_api.go      JSON API handlers ──┘
    │
    └── SSE event emission (via middleware/wrapper)
```

**Key constraint:** HTMX handlers in `serve.go` remain **completely untouched**. Shared business logic is extracted into common functions that both `serve.go` and `serve_api.go` call.

## Process

### Step 1: Endpoint Structure in `serve_api.go`

All JSON API endpoints live in `serve_api.go` under the `/api/v1/` prefix.

```go
// serve_api.go

func (s *Server) registerAPIRoutes() {
    // Entity browsers (M1)
    s.mux.HandleFunc("GET /api/v1/agents", s.apiListAgents)
    s.mux.HandleFunc("GET /api/v1/skills", s.apiListSkills)
    s.mux.HandleFunc("GET /api/v1/skills/{name}", s.apiGetSkill)
    s.mux.HandleFunc("GET /api/v1/snippets", s.apiListSnippets)
    s.mux.HandleFunc("GET /api/v1/snippets/{lang}", s.apiListSnippetsByLang)

    // Workspaces (M2)
    s.mux.HandleFunc("GET /api/v1/workspaces", s.apiListWorkspaces)
    s.mux.HandleFunc("GET /api/v1/workspaces/{name}", s.apiGetWorkspace)
    s.mux.HandleFunc("POST /api/v1/workspaces", s.apiCreateWorkspace)

    // SSE (M0)
    s.mux.HandleFunc("GET /api/v1/events/stream", s.apiEventStream)

    // ... additional endpoints per milestone
}
```

### Step 2: JSON Handler Pattern

Every JSON API handler follows the same pattern:

```go
func (s *Server) apiListWorkspaces(w http.ResponseWriter, r *http.Request) {
    workspaces, err := s.listWorkspaces(r.Context())
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workspaces")
        return
    }
    writeJSON(w, http.StatusOK, workspaces)
}
```

**Helper functions:**

```go
func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error": message,
        "code":  code,
    })
}
```

### Step 3: JSON Error Response Format

All error responses use a consistent format:

```json
{
  "error": "Human-readable error message",
  "code": "MACHINE_READABLE_CODE"
}
```

**Standard error codes:**

| HTTP Status | Code | When |
|-------------|------|------|
| 400 | `BAD_REQUEST` | Invalid input, missing required fields |
| 404 | `NOT_FOUND` | Resource doesn't exist |
| 409 | `CONFLICT` | Resource already exists, concurrent modification |
| 422 | `VALIDATION_ERROR` | Input fails validation rules |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

### Step 4: Shared Business Logic Extraction

Extract common functions from `serve.go` handlers. Both HTMX and JSON handlers call the same shared functions.

**Before (logic in HTMX handler):**

```go
// serve.go - HTMX handler
func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
    workspaces, err := s.store.ListWorkspaces(r.Context())
    if err != nil {
        http.Error(w, "internal error", 500)
        return
    }
    // ... render HTML template
}
```

**After (shared function):**

```go
// business_logic.go (or inline in server struct methods)
func (s *Server) listWorkspaces(ctx context.Context) ([]Workspace, error) {
    return s.store.ListWorkspaces(ctx)
}

// serve.go - HTMX handler (calls shared function)
func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
    workspaces, err := s.listWorkspaces(r.Context())
    if err != nil {
        http.Error(w, "internal error", 500)
        return
    }
    // ... render HTML template
}

// serve_api.go - JSON handler (calls same shared function)
func (s *Server) apiListWorkspaces(w http.ResponseWriter, r *http.Request) {
    workspaces, err := s.listWorkspaces(r.Context())
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workspaces")
        return
    }
    writeJSON(w, http.StatusOK, workspaces)
}
```

**Rule:** Never reimplement business logic. Always extract and share.

### Step 5: SSE Event Emission (R-4, R-20)

SSE events must be emitted structurally, not via manual `publish()` calls scattered in handlers. Use a wrapper/middleware pattern.

**Pattern: Deferred event publishing after transaction commit**

```go
func (s *Server) apiCreateWorkspace(w http.ResponseWriter, r *http.Request) {
    var req CreateWorkspaceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
        return
    }

    workspace, err := s.createWorkspace(r.Context(), req)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workspace")
        return
    }

    // Emit SSE event AFTER successful mutation (R-20)
    s.publishSSE("workspace:update", workspace)

    writeJSON(w, http.StatusCreated, workspace)
}
```

**Structural middleware approach (preferred for R-4):**

```go
func (s *Server) withSSEEmission(eventType string, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Wrap response writer to capture status
        rw := &responseCapture{ResponseWriter: w}
        next(rw, r)

        // Only emit on success (2xx status codes)
        if rw.statusCode >= 200 && rw.statusCode < 300 {
            s.publishSSE(eventType, rw.body)
        }
    }
}

// Registration
s.mux.HandleFunc("POST /api/v1/workspaces",
    s.withSSEEmission("workspace:update", s.apiCreateWorkspace))
```

**Rules:**
- SSE events are emitted AFTER the mutation succeeds, never before or during
- Use structural middleware so developers can't forget to emit events
- The middleware wraps the handler and publishes only on 2xx responses

### Step 6: Idempotent Endpoints (R-10)

Mutation endpoints must be idempotent. Repeating the same request produces the same result.

**Example: Start session**

```go
func (s *Server) apiStartSession(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")

    session, err := s.getSession(r.Context(), name)
    if err != nil {
        writeJSONError(w, http.StatusNotFound, "NOT_FOUND", "workspace not found")
        return
    }

    // Idempotent: if already running, return current state (R-10)
    if session.Status == "running" {
        writeJSON(w, http.StatusOK, session)
        return
    }

    session, err = s.startSession(r.Context(), name)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start session")
        return
    }

    s.publishSSE("session:update", session)
    writeJSON(w, http.StatusOK, session)
}
```

**Idempotency rules:**
- `POST /start` on running session: return current state (200), don't error
- `POST /stop` on stopped session: return current state (200), don't error
- `POST /respond` with stale question ID: return 409 with "Question expired" (R-21)
- `POST /create` with existing name: return 409 with "Already exists"

### Step 7: SPA Catch-All Handler (R-12, R-23)

The Go router must serve `index.html` for all SPA routes when `sgai-ui=react`, but **explicitly exclude** `/api/v1/*` routes.

```go
func (s *Server) registerRoutes() {
    // 1. API routes FIRST (R-23)
    s.registerAPIRoutes()

    // 2. Static assets
    s.mux.Handle("/assets/", http.FileServer(s.webappFS))

    // 3. SPA catch-all LAST
    s.mux.HandleFunc("/", s.handleSPACatchAll)
}

func (s *Server) handleSPACatchAll(w http.ResponseWriter, r *http.Request) {
    // Check cookie
    cookie, _ := r.Cookie("sgai-ui")
    if cookie == nil || cookie.Value != "react" {
        // Serve HTMX as before
        s.handleHTMX(w, r)
        return
    }

    // Serve React index.html for all non-API, non-static routes
    f, err := s.webappFS.Open("dist/index.html")
    if err != nil {
        http.Error(w, "React app not found", 500)
        return
    }
    defer f.Close()
    w.Header().Set("Content-Type", "text/html")
    io.Copy(w, f)
}
```

**Critical:** `/api/v1/*` routes are registered BEFORE the SPA catch-all. The catch-all must never intercept API requests.

## Rules

1. **Never modify `serve.go`** — HTMX handlers stay untouched. Extract shared logic; don't move or change existing handlers.

2. **Share business logic, don't reimplement** — Both JSON and HTMX handlers call the same extracted functions. Never write the same logic twice.

3. **Consistent JSON error format** — Every error response uses `{"error": "...", "code": "..."}` format.

4. **SSE events after commit only** — Events are published after the mutation succeeds. Use structural middleware to enforce this (R-4, R-20).

5. **Idempotent mutations** — Repeating the same mutation request returns success with current state, not an error (R-10).

6. **API routes before catch-all** — `/api/v1/*` routes must be registered before the SPA catch-all handler (R-23).

## Checklist

Before completing API endpoint work, verify:

- [ ] Endpoint registered under `/api/v1/` prefix
- [ ] Handler calls shared business logic (not reimplemented)
- [ ] Error responses use `{"error": "...", "code": "..."}` format
- [ ] SSE event emitted after successful mutation
- [ ] Mutation endpoints are idempotent
- [ ] `serve.go` is unchanged
- [ ] API routes registered before SPA catch-all

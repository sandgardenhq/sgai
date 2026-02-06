---
name: go-http-server-routes
description: Go 1.22+ enhanced HTTP routing with method matching and wildcards. When defining HTTP routes using net/http ServeMux in Go 1.22+
metadata:
    languages: go
---

# Go HTTP Server Routes (Go 1.22+)

Enhanced routing patterns for `net/http.ServeMux` introduced in Go 1.22.

## Reference URLs

- https://go.dev/blog/routing-enhancements - Official announcement
- https://pkg.go.dev/net/http#ServeMux - ServeMux documentation

## Method Matching

Specify HTTP methods directly in patterns:

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /posts/{id}", getPost)
mux.HandleFunc("POST /posts", createPost)
mux.HandleFunc("PUT /posts/{id}", updatePost)
mux.HandleFunc("DELETE /posts/{id}", deletePost)
```

- `GET` also matches `HEAD` requests
- All other methods match exactly
- Unmatched methods return `405 Method Not Allowed` with `Allow` header

## Wildcards

### Single Segment: `{name}`

Matches exactly one path segment:

```go
mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

### Remaining Segments: `{name...}`

Matches all remaining path segments:

```go
mux.HandleFunc("GET /files/{pathname...}", func(w http.ResponseWriter, r *http.Request) {
    pathname := r.PathValue("pathname")
    // /files/docs/readme.txt -> pathname = "docs/readme.txt"
})
```

### Exact Trailing Slash: `{$}`

Matches only the path with trailing slash:

```go
mux.HandleFunc("GET /posts/{$}", listPosts)   // matches /posts/ only
mux.HandleFunc("GET /posts/{id}", getPost)    // matches /posts/123
```

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `/posts/{$}` | `/posts/` | `/posts`, `/posts/123` |
| `/posts/` | `/posts/`, `/posts/123`, `/posts/a/b` | `/posts` |
| `/posts/{id}` | `/posts/123` | `/posts/`, `/posts/a/b` |

## PathValue API

Extract wildcard values from requests:

```go
func getPost(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    // use id...
}
```

For third-party routers to integrate:

```go
r.SetPathValue("id", "123")
```

## Precedence Rules

The **most specific pattern wins**. A pattern is more specific if it matches a strict subset of requests.

### Examples

| Pattern A | Pattern B | Winner | Reason |
|-----------|-----------|--------|--------|
| `/posts/latest` | `/posts/{id}` | A | Literal matches fewer requests |
| `GET /posts/{id}` | `/posts/{id}` | A | Method constraint is more specific |
| `/users/{u}/posts/latest` | `/users/{u}/posts/{id}` | A | Literal segment more specific |

### Conflicts

Two patterns conflict if neither is more specific than the other. Registering conflicting patterns causes a **panic**.

```go
// CONFLICT - will panic
mux.HandleFunc("/posts/{id}", handler1)
mux.HandleFunc("/{resource}/latest", handler2)
// Both match /posts/latest, neither is more specific
```

### Host Precedence

Patterns with hosts take precedence over patterns without:

```go
mux.HandleFunc("example.com/", hostHandler)  // wins for example.com
mux.HandleFunc("/", defaultHandler)          // wins for other hosts
```

## Complete Example

```go
package main

import (
    "encoding/json"
    "net/http"
)

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /api/posts", listPosts)
    mux.HandleFunc("GET /api/posts/{id}", getPost)
    mux.HandleFunc("POST /api/posts", createPost)
    mux.HandleFunc("PUT /api/posts/{id}", updatePost)
    mux.HandleFunc("DELETE /api/posts/{id}", deletePost)

    mux.HandleFunc("GET /static/{filepath...}", serveStatic)

    http.ListenAndServe(":8080", mux)
}

func getPost(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    post, err := findPost(id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(post)
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
    filepath := r.PathValue("filepath")
    http.ServeFile(w, r, "static/"+filepath)
}
```

## Common Patterns

### Resource CRUD

```go
mux.HandleFunc("GET /api/{resource}", listHandler)
mux.HandleFunc("GET /api/{resource}/{id}", getHandler)
mux.HandleFunc("POST /api/{resource}", createHandler)
mux.HandleFunc("PUT /api/{resource}/{id}", updateHandler)
mux.HandleFunc("DELETE /api/{resource}/{id}", deleteHandler)
```

### Nested Resources

```go
mux.HandleFunc("GET /users/{userID}/posts", listUserPosts)
mux.HandleFunc("GET /users/{userID}/posts/{postID}", getUserPost)
```

### Special Routes Before Wildcards

```go
mux.HandleFunc("GET /posts/latest", getLatestPost)  // more specific
mux.HandleFunc("GET /posts/{id}", getPost)          // less specific
```

## Compatibility

For code that relied on pre-1.22 behavior (literal braces in patterns):

```bash
GODEBUG=httpmuxgo121=1 ./myapp
```

## Common Mistakes

- **Forgetting method prefix** - `/posts/{id}` matches all methods
- **Conflicting patterns** - causes panic at registration, not runtime
- **Missing `{$}`** - `/posts/` matches `/posts/anything`, use `/posts/{$}` for exact match
- **Wildcard naming** - `{id}` and `{identifier}` are different wildcards, name doesn't affect precedence

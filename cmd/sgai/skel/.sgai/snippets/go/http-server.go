---
name: HTTP Server
description: Simple HTTP server that responds with Hello World
when_to_use: When creating a basic web server in Go
---

/* A simple HTTP server in Go */
package main

import (
    "fmt"
    "net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}

func main() {
    http.HandleFunc("/", helloHandler)
    fmt.Println("Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}
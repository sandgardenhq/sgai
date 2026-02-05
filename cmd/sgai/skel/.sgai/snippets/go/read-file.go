---
name: Read File
description: Read the contents of a file into a string; When you need to read an entire file in Go
---

/* Read the entire contents of a file into a string */
package main

import (
    "fmt"
    "os"
)

func main() {
    data, err := os.ReadFile("example.txt")
    if err != nil {
        fmt.Println("Error reading file:", err)
        return
    }
    fmt.Println(string(data))
}

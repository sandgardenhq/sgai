---
name: JSON Encode/Marshal
description: YOU MUST USE json.Encoder to encode data to JSON format. This is the mandatory pattern for encoding JSON in this codebase.
---

package main

import (
	"bytes"
	"encoding/json"
	"log"
)

func main() {
	// data _is_ the information that needs to be converted to JSON
	var data = struct{}{}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		log.Fatal("cannot encode data", err)
		// or
		// return fmt.Errorf("cannot encode data: %w", err)
	}

	fmt.Println("use encoded JSON data", buf.String())
}

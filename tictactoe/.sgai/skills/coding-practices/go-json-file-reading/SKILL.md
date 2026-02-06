---
name: go-json-file-reading
description: Use when reading JSON files in Go applications, especially for configuration files, data files, or any JSON parsing that requires comprehensive error handling for missing files, invalid JSON, and permission errors
---

# Go JSON File Reading

## Overview

This skill provides comprehensive patterns for reading JSON files in Go with proper error handling, context wrapping, and best practices. It addresses common failure modes like missing files, invalid JSON syntax, permission errors, and provides structured error messages for debugging.

## When to Use

- Use when reading configuration files (config.json, settings.json)
- Use when reading data files for application initialization
- Use when parsing JSON responses saved to disk
- Use when any JSON file reading requires robust error handling
- Don't use for simple JSON string parsing (use json.Unmarshal directly)
- Don't use when reading from network (use HTTP client patterns)

## Process

### Step 1: Define the Target Struct
- [ ] Create Go struct that matches JSON structure
- [ ] Add JSON tags for proper field mapping
- [ ] Include pointer types for optional fields
- [ ] Consider using `omitempty` for optional fields

### Step 2: Implement File Reading with Error Handling
- [ ] Use `os.ReadFile()` for simple cases or `os.Open()` for streaming
- [ ] Handle file not found errors specifically
- [ ] Handle permission errors with clear messaging
- [ ] Wrap errors with context using `fmt.Errorf()` or custom error types

### Step 3: Parse JSON with Validation
- [ ] Use `json.Unmarshal()` for in-memory parsing
- [ ] Handle JSON syntax errors with line/column information
- [ ] Validate required fields after parsing
- [ ] Provide meaningful error messages for validation failures

### Step 4: Add Context and Logging
- [ ] Wrap errors with file path and operation context
- [ ] Add structured logging for debugging
- [ ] Consider using `context.Context` for cancellation
- [ ] Provide fallback values where appropriate

## Rules

1. **Always handle file-not-found separately** - Don't treat missing config the same as invalid JSON
2. **Never ignore permission errors** - These require different user action than syntax errors
3. **Wrap errors with context** - Always include file path and operation in error messages
4. **Validate after parsing** - JSON syntax ≠ valid data structure
5. **Use appropriate error types** - Distinguish between system errors and data errors
6. **Provide actionable error messages** - User should know exactly what to fix

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "Just return the error, user will figure it out" | Users need context to fix issues. "File not found" vs "Invalid JSON at line 5" require different actions. |
| "Error handling makes code complex" | Proper error handling prevents runtime panics and makes debugging possible. Complexity is necessary for reliability. |
| "I'll just log and continue" | Silent failures create mysterious bugs. Errors should be explicit and handled appropriately. |
| "The JSON will always be valid" | Files get corrupted, users make mistakes, systems fail. Assume failure, handle it gracefully. |
| "Permission errors are rare" | When they happen, they're blocking. Different error handling than syntax errors. |

## Red Flags - STOP

- "I'll just use `json.Unmarshal` directly without file checks"
- "Let the error bubble up without context"
- "Assume the file exists and is valid JSON"
- "Ignore permission errors - they won't happen"
- "Use panic for error handling"
- "Return nil instead of proper error"

## Examples

### Good Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type Config struct {
    DatabaseURL string `json:"database_url"`
    Port        int    `json:"port"`
    Debug       *bool  `json:"debug,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
    // Step 1: Read file with specific error handling
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("config file not found: %s", path)
        }
        if os.IsPermission(err) {
            return nil, fmt.Errorf("permission denied reading config file: %s", path)
        }
        return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
    }

    // Step 2: Parse JSON with syntax error handling
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        if syntaxErr, ok := err.(*json.SyntaxError); ok {
            return nil, fmt.Errorf("invalid JSON syntax in config file %s at offset %d: %w", 
                path, syntaxErr.Offset, err)
        }
        if unmarshalErr, ok := err.(*json.UnmarshalTypeError); ok {
            return nil, fmt.Errorf("invalid JSON type in config file %s at field %s: expected %s, got %s", 
                path, unmarshalErr.Field, unmarshalErr.Type, unmarshalErr.Value)
        }
        return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
    }

    // Step 3: Validate required fields
    if config.DatabaseURL == "" {
        return nil, fmt.Errorf("config file %s: missing required field 'database_url'", path)
    }
    if config.Port == 0 {
        return nil, fmt.Errorf("config file %s: missing required field 'port'", path)
    }

    return &config, nil
}
```

### Bad Example

```go
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err // No context, no specific handling
    }

    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err // No syntax error details
    }

    return &config, nil // No validation of required fields
}
```

**Why this is wrong:**
- No specific error handling for missing vs permission vs other errors
- No context in error messages (which file? what operation?)
- No validation of required fields after parsing
- No distinction between syntax errors and type errors

## Checklist

Before completing, verify:

- [ ] File reading handles not found, permission, and other errors separately
- [ ] JSON parsing provides syntax error details (line/offset when possible)
- [ ] All errors include file path and operation context
- [ ] Required fields are validated after parsing
- [ ] Error messages are actionable and specific
- [ ] Optional fields use pointers or have default values
- [ ] No silent failures or ignored errors
- [ ] Struct fields have proper JSON tags
- [ ] Error types distinguish between system and data errors

## Modern Go Features for JSON Handling

### Generic JSON Loading (Go 1.18+)

Create reusable JSON loading functions using generics:

```go
// Generic JSON file loader
func LoadJSON[T any](path string) (*T, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("file not found: %s", path)
        }
        return nil, fmt.Errorf("reading %s: %w", path, err)
    }
    
    var result T
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, fmt.Errorf("parsing JSON in %s: %w", path, err)
    }
    
    return &result, nil
}

// Usage
config, err := LoadJSON[Config]("config.json")
users, err := LoadJSON[[]User]("users.json")
```

### Validating JSON Results with slices/maps

Use modern stdlib for validation:

```go
import (
    "maps"
    "slices"
)

func ValidateConfig(cfg *Config) error {
    // Check required values are in allowed list
    allowedEnvs := []string{"dev", "staging", "prod"}
    if !slices.Contains(allowedEnvs, cfg.Environment) {
        return fmt.Errorf("invalid environment: %s", cfg.Environment)
    }
    
    // Check for duplicate keys in loaded data
    seen := make(map[string]bool)
    for _, item := range cfg.Items {
        if seen[item.ID] {
            return fmt.Errorf("duplicate item ID: %s", item.ID)
        }
        seen[item.ID] = true
    }
    
    return nil
}
```

### Generic Config with Defaults

```go
// Config with generic defaults support
type ConfigLoader[T any] struct {
    defaults T
}

func NewConfigLoader[T any](defaults T) *ConfigLoader[T] {
    return &ConfigLoader[T]{defaults: defaults}
}

func (l *ConfigLoader[T]) Load(path string) (*T, error) {
    // Start with defaults
    result := l.defaults
    
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            // Return defaults if file doesn't exist
            return &result, nil
        }
        return nil, err
    }
    
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// Usage
loader := NewConfigLoader(Config{Port: 8080, Debug: false})
cfg, err := loader.Load("config.json")
```

**References:**
- https://go.dev/doc/tutorial/generics
- https://pkg.go.dev/slices
- https://pkg.go.dev/maps
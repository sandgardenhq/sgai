---
name: opencode-plugin-development
description: Guide for creating and implementing plugins to extend OpenCode functionality, including hooks, custom tools, and event handling. When you need to add custom functionality to OpenCode, such as notifications, custom tools, or modifying behavior.
---

# OpenCode Plugin Development

## Overview

Plugins allow you to extend OpenCode by hooking into various events, adding custom tools, and customizing behavior. This skill guides you through creating plugins using JavaScript/TypeScript modules that export plugin functions.

## When to Use

- Adding new features or integrations
- Modifying OpenCode's default behavior
- Implementing custom tools
- Handling specific events (e.g., notifications)
- Protecting sensitive files

## Plugin Structure

### Basic Plugin Format

A plugin is a JavaScript/TypeScript module that exports one or more plugin functions. Each function receives a context object and returns a hooks object.

```javascript
export const MyPlugin = async ({ project, client, $, directory, worktree }) => {
  console.log("Plugin initialized!")
  return {
    // Hook implementations go here
  }
}
```

### Context Parameters

- `project`: Current project information
- `directory`: Current working directory
- `worktree`: Git worktree path
- `client`: OpenCode SDK client for AI interactions
- `$`: Bun's shell API for executing commands

### TypeScript Support

```typescript
import type { Plugin } from "@opencode-ai/plugin"

export const MyPlugin: Plugin = async ({ project, client, $, directory, worktree }) => {
  return {
    // Type-safe hook implementations
  }
}
```

## Plugin Location

Plugins are loaded from:
1. `.sgai/plugin` directory in your project
2. Or globally in `~/.config/opencode/plugin`

## Available Hooks

### Event Hook
Listen for OpenCode events:

```javascript
return {
  event: async ({ event }) => {
    if (event.type === "session.idle") {
      await $`osascript -e 'display notification "Session completed!" with title "opencode"'`
    }
  }
}
```

### Tool Hook
Add custom tools:

```typescript
import { type Plugin, tool } from "@opencode-ai/plugin"

export const CustomToolsPlugin: Plugin = async (ctx) => {
  return {
    tool: {
      mytool: tool({
        description: "This is a custom tool",
        args: {
          foo: tool.schema.string(),
        },
        async execute(args, ctx) {
          return `Hello ${args.foo}!`
        },
      }),
    },
  }
}
```

### Tool Execution Hooks
Intercept tool execution:

```javascript
return {
  "tool.execute.before": async (input, output) => {
    if (input.tool === "read" && output.args.filePath.includes(".env")) {
      throw new Error("Do not read .env files")
    }
  }
}
```

## Development Process

1. **Plan the Plugin**: Determine what functionality to add and which hooks to use
2. **Create Plugin File**: Write the plugin module in `.sgai/plugin/`
3. **Implement Hooks**: Add the necessary hook functions
4. **Test Plugin**: Load and test in OpenCode
5. **Handle Errors**: Add proper error handling and logging

## Best Practices

- Use TypeScript for better development experience
- Handle errors gracefully in hooks
- Test plugins thoroughly before deployment
- Document custom tools clearly
- Follow OpenCode's plugin conventions

## Common Patterns

### Notification Plugin
Send notifications on events:

```javascript
export const NotificationPlugin = async ({ $ }) => {
  return {
    event: async ({ event }) => {
      if (event.type === "session.idle") {
        await $`notify-send "OpenCode" "Session completed"`
      }
    }
  }
}
```

### File Protection Plugin
Prevent access to sensitive files:

```javascript
export const FileProtection = async () => {
  return {
    "tool.execute.before": async (input) => {
      const protectedPatterns = [/\.env$/, /secrets\//, /private\//]
      if (input.tool === "read" && protectedPatterns.some(p => p.test(input.args.filePath))) {
        throw new Error("Access denied to protected file")
      }
    }
  }
}
```

### Custom Tool Plugin
Add domain-specific tools:

```typescript
export const DomainTools: Plugin = async () => {
  return {
    tool: {
      analyzeCode: tool({
        description: "Analyze code for specific patterns",
        args: {
          filePath: tool.schema.string(),
          pattern: tool.schema.string()
        },
        execute: async (args) => {
          // Implementation
          return "Analysis complete"
        }
      })
    }
  }
}
```

## Troubleshooting

- Ensure plugin files are in the correct location
- Check console logs for initialization errors
- Verify hook signatures match the expected types
- Test custom tools with simple inputs first
- Use the OpenCode client for AI interactions when needed

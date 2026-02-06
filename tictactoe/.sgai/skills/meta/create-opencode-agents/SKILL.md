---
name: create-opencode-agents
description: Guide for creating custom opencode agents with proper configuration, tools, permissions, and reasoning for specific tasks. When creating new opencode agents for security auditing, code review, planning, documentation, or specialized workflows; when built-in agents don't suffice; when configuring agents with restricted tools or custom prompts
---

# Create OpenCode Agents

## Overview

OpenCode agents are specialized AI assistants configured for specific tasks with custom prompts, models, tools, and permissions. Creating effective agents requires understanding the configuration options and applying them appropriately to the agent's purpose.

## When to Use

Use this skill when:
- You need a new agent for a specialized task like security auditing or code review
- Built-in agents (Build, Plan, General) don't meet your requirements
- You need to restrict tools for safety or focus
- You want custom prompts or models for specific workflows
- You're configuring agents for team use or automation

Don't use for:
- Simple tasks that built-in agents can handle
- When you just need to modify existing agent configs

## Core Pattern

1. **Define Purpose**: Clearly state what the agent does and when to use it
2. **Research Options**: Review available configuration options from docs
3. **Choose Configuration**: Select appropriate model, tools, permissions, prompt
4. **Create Config**: Write the agent configuration file
5. **Test and Refine**: Verify the agent works as intended

## Implementation

### Step 1: Define Agent Purpose

Start with a clear description of what the agent does:

```yaml
description: Performs security audits and identifies vulnerabilities
```

Include when to use it in the description for better discoverability.

### Step 2: Research Configuration Options

Key options from OpenCode docs:

- **mode**: "primary" (main agent) or "subagent" (specialized helper)
- **model**: Choose based on task (faster for planning, more capable for complex work)
- **temperature**: 0.0-0.2 for focused tasks, 0.6-1.0 for creative
- **tools**: Enable/disable specific tools (write, edit, bash, etc.)
- **permissions**: Control tool usage ("allow", "ask", "deny")
- **prompt**: Custom system prompt file

### Step 3: Configure for Task

For security auditing agents:
- Use subagent mode
- Restrict destructive tools (write: false, edit: false)
- Allow read-only tools (grep, read, list)
- Set permissions to deny risky operations
- Use focused prompt emphasizing security analysis

### Step 4: Create Configuration File

Create markdown file in `.sgai/agent/` (project) or `~/.config/opencode/agent/` (global):

```markdown
---
description: Performs security audits and identifies vulnerabilities
mode: subagent
temperature: 0.1
tools:
  write: false
  edit: false
  bash: false
permissions:
  edit: deny
  bash: deny
  webfetch: allow
---

You are a security expert specializing in code analysis. Focus on identifying potential security issues including:
- Input validation vulnerabilities
- Authentication and authorization flaws
- Data exposure risks
- Dependency vulnerabilities
- Configuration security issues

Provide detailed analysis with specific code locations and severity levels. Do not make changes to code.
```

### Step 5: Test Configuration

- Invoke the agent with @mention
- Test with sample code or scenarios
- Verify tool restrictions work
- Refine prompt and config as needed

## Common Mistakes

- **Over-restricting**: Denying too many tools makes agent useless
- **Under-restricting**: Allowing dangerous tools in security contexts
- **Vague descriptions**: Agents won't know when to invoke subagents
- **No custom prompt**: Agent behaves like general assistant
- **Wrong mode**: Primary agents for everything clutters interface

## Quick Reference

| Task Type | Mode | Tools | Permissions |
|-----------|------|-------|-------------|
| Security Audit | subagent | read-only | deny edit/bash |
| Code Review | subagent | read-only | deny edit |
| Planning | primary | restricted | ask for changes |
| Documentation | subagent | write allowed | allow file ops |

## Real-World Impact

Properly configured agents:
- Reduce security risks through restricted tools
- Improve efficiency with task-specific prompts
- Enable better collaboration with specialized roles
- Prevent accidental changes in analysis contexts

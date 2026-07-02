---
name: create-opencode-agents
description: Guide for creating custom opencode agents with proper configuration, permissions, path-based delegation, work splitting, and strict status envelopes. Use when creating or updating agents for coordination, implementation, review, deployment, verification, documentation, or specialized workflows.
---

# Create OpenCode Agents

## Overview

OpenCode agents are specialized AI assistants configured for specific tasks with custom prompts, models, tools, permissions, and message contracts. Creating effective SGAI agents requires bounded context packets, explicit work splitting, and strict return envelopes.

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
4. **Define Protocol**: Add packet intake, splitting rules, and return envelopes
5. **Create Config**: Write the agent configuration file
6. **Test and Refine**: Verify the agent works as intended

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
- **disable**: Set `true` only when an agent configuration must be temporarily unavailable
- **hidden**: Hide internal subagents from user autocomplete
- **permissions**: Control tool usage (`allow`, `ask`, `deny`, or path/command maps)
- **prompt**: Custom system prompt file

### Step 3: Configure for Task

For security auditing agents:
- Use subagent mode
- Restrict destructive tools (write: false, edit: false)
- Allow read-only tools (grep, read, list)
- Set permissions to deny risky operations
- Use focused prompt emphasizing security analysis

For SGAI wrapper agents:
- Use `mode: primary`
- Deny direct edits in the prompt; do not add parent `permission.edit: deny` if child developer subagents must edit
- Allow only the developer/reviewer subagents that wrapper may invoke through `permission.task`
- Require every delegated task to use a bounded `TASK_PACKET`
- Require a `WORK_SPLIT` map before broad fanout
- Reconcile developer/reviewer envelopes before reporting completion to the coordinator

For developer subagents:
- Use `mode: subagent` and usually `hidden: true`
- Allow edit tools only when the role actually changes files
- Read packet paths first; return `NEEDS_CONTEXT` instead of rediscovering broadly
- Return the strict `STATUS` envelope

For reviewer/verifier subagents:
- Use `mode: subagent`, usually `hidden: true`, and read-only permissions
- Review only the provided scope unless discovery is explicitly requested
- Return the strict `REVIEW_STATUS` envelope

For forbidden flows:
- Delete the agent file
- Remove references to it from prompts, skills, templates, examples, and GOAL composers
- Do not leave empty replacement agent files

### Step 4: Create Configuration File

Create markdown file in `.sgai/agent/` (project) or `~/.config/opencode/agent/` (global). For skeleton agents, edit `cmd/sgai/skel/.sgai/agent/`; for project overlays, edit `sgai/agent/` with the complete replacement file.

```markdown
---
description: Performs security audits and identifies vulnerabilities
mode: subagent
temperature: 0.1
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
- **No packet protocol**: Subagents waste context rediscovering the repository
- **No split map**: Coordinators parallelize unsafe work or serialize independent work without reason
- **No return envelope**: Callers cannot reconcile completion, blockers, and verification evidence
- **Legacy routing**: Agents call workflow-state tools or write project-management files despite denied permissions

## Required Message Contracts

Every delegating agent must know how to split, delegate, and reconcile work. Coordinator and wrapper-agent prompts must load and follow `subagent-delegation-protocol` skill for Task Packets, Work Splits, parallelization, and reconciliation.

Developer/execution agents return:

Load and follow `subagent-return-protocol` skill, then return its Execution Envelope exactly.

Reviewer/verifier agents return:

Load and follow `subagent-return-protocol` skill, then return its Review Envelope exactly.

## Splitting and Reconciliation Rules

Use `subagent-delegation-protocol` skill for splitting and reconciliation rules. Do not duplicate the packet, split-map, or reconciliation schema in each agent prompt.

## Quick Reference

| Task Type | Mode | Tools | Permissions |
|-----------|------|-------|-------------|
| Security Audit | subagent | read-only | deny edit/bash |
| Code Review | subagent | read-only | deny edit |
| Planning | primary | restricted | ask for changes |
| Documentation | subagent | write allowed | allow file ops |
| Front Wrapper | primary | task only | allow specific developer/reviewer subagents |
| Developer | subagent | edit allowed | deny task unless explicitly needed |
| Reviewer/Verifier | subagent | read-only | deny edit |
| Forbidden Workflow | delete file | none | deny all |

## Real-World Impact

Properly configured agents:
- Reduce security risks through restricted tools
- Improve efficiency with task-specific prompts
- Enable better collaboration with specialized roles
- Prevent accidental changes in analysis contexts

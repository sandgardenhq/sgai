<overview>
Architectural patterns for building prompt-native agent systems. These patterns emerge from the philosophy that features should be defined in prompts, not code, and that tools should be primitives.
</overview>

<pattern name="event-driven-agent">
## Event-Driven Agent Architecture

The agent runs as a long-lived process that responds to events. Events become prompts.

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent Loop                                │
├─────────────────────────────────────────────────────────────┤
│  Event Source → Agent (Claude) → Tool Calls → Response      │
└─────────────────────────────────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌─────────┐    ┌──────────┐    ┌───────────┐
    │ Content │    │   Self   │    │   Data    │
    │  Tools  │    │  Tools   │    │   Tools   │
    └─────────┘    └──────────┘    └───────────┘
    (write_file)   (read_source)   (store_item)
                   (restart)       (list_items)
```

**Key characteristics:**
- Events (messages, webhooks, timers) trigger agent turns
- Agent decides how to respond based on system prompt
- Tools are primitives for IO, not business logic
- State persists between events via data tools

**Example: Discord feedback bot**
```typescript
// Event source
client.on("messageCreate", (message) => {
  if (!message.author.bot) {
    runAgent({
      userMessage: `New message from ${message.author}: "${message.content}"`,
      channelId: message.channelId,
    });
  }
});

// System prompt defines behavior
const systemPrompt = `
When someone shares feedback:
1. Acknowledge their feedback warmly
2. Ask clarifying questions if needed
3. Store it using the feedback tools
4. Update the feedback site

Use your judgment about importance and categorization.
`;
```
</pattern>

<pattern name="two-layer-git">
## Two-Layer Git Architecture

For self-modifying agents, separate code (shared) from data (instance-specific).

```
┌─────────────────────────────────────────────────────────────┐
│                     GitHub (shared repo)                     │
│  - src/           (agent code)                              │
│  - site/          (web interface)                           │
│  - package.json   (dependencies)                            │
│  - .gitignore     (excludes data/, logs/)                   │
└─────────────────────────────────────────────────────────────┘
                          │
                     git clone
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  Instance (Server)                           │
│                                                              │
│  FROM GITHUB (tracked):                                      │
│  - src/           → pushed back on code changes             │
│  - site/          → pushed, triggers deployment             │
│                                                              │
│  LOCAL ONLY (untracked):                                     │
│  - data/          → instance-specific storage               │
│  - logs/          → runtime logs                            │
│  - .env           → secrets                                 │
└─────────────────────────────────────────────────────────────┘
```

**Why this works:**
- Code and site are version controlled (GitHub)
- Raw data stays local (instance-specific)
- Site is generated from data, so reproducible
- Automatic rollback via git history
</pattern>

<pattern name="multi-instance">
## Multi-Instance Branching

Each agent instance gets its own branch while sharing core code.

```
main                        # Shared features, bug fixes
├── instance/feedback-bot   # Every Reader feedback bot
├── instance/support-bot    # Customer support bot
└── instance/research-bot   # Research assistant
```

**Change flow:**
| Change Type | Work On | Then |
|-------------|---------|------|
| Core features | main | Merge to instance branches |
| Bug fixes | main | Merge to instance branches |
| Instance config | instance branch | Done |
| Instance data | instance branch | Done |

**Sync tools:**
```typescript
tool("self_deploy", "Pull latest from main, rebuild, restart", ...)
tool("sync_from_instance", "Merge from another instance", ...)
tool("propose_to_main", "Create PR to share improvements", ...)
```
</pattern>

<pattern name="site-as-output">
## Site as Agent Output

The agent generates and maintains a website as a natural output, not through specialized site tools.

```
Discord Message
      ↓
Agent processes it, extracts insights
      ↓
Agent decides what site updates are needed
      ↓
Agent writes files using write_file primitive
      ↓
Git commit + push triggers deployment
      ↓
Site updates automatically
```

**Key insight:** Don't build site generation tools. Give the agent file tools and teach it in the prompt how to create good sites.

```markdown
## Site Management

You maintain a public feedback site. When feedback comes in:
1. Use write_file to update site/public/content/feedback.json
2. If the site's React components need improvement, modify them
3. Commit changes and push to trigger Vercel deploy

The site should be:
- Clean, modern dashboard aesthetic
- Clear visual hierarchy
- Status organization (Inbox, Active, Done)

You decide the structure. Make it good.
```
</pattern>

<pattern name="approval-gates">
## Approval Gates Pattern

Separate "propose" from "apply" for dangerous operations.

```typescript
// Pending changes stored separately
const pendingChanges = new Map<string, string>();

tool("write_file", async ({ path, content }) => {
  if (requiresApproval(path)) {
    // Store for approval
    pendingChanges.set(path, content);
    const diff = generateDiff(path, content);
    return {
      text: `Change requires approval.\n\n${diff}\n\nReply "yes" to apply.`
    };
  } else {
    // Apply immediately
    writeFileSync(path, content);
    return { text: `Wrote ${path}` };
  }
});

tool("apply_pending", async () => {
  for (const [path, content] of pendingChanges) {
    writeFileSync(path, content);
  }
  pendingChanges.clear();
  return { text: "Applied all pending changes" };
});
```

**What requires approval:**
- src/*.ts (agent code)
- package.json (dependencies)
- system prompt changes

**What doesn't:**
- data/* (instance data)
- site/* (generated content)
- docs/* (documentation)
</pattern>

<design_questions>
## Questions to Ask When Designing

1. **What events trigger agent turns?** (messages, webhooks, timers, user requests)
2. **What primitives does the agent need?** (read, write, call API, restart)
3. **What decisions should the agent make?** (format, structure, priority, action)
4. **What decisions should be hardcoded?** (security boundaries, approval requirements)
5. **How does the agent verify its work?** (health checks, build verification)
6. **How does the agent recover from mistakes?** (git rollback, approval gates)
</design_questions>

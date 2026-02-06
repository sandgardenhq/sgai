<overview>
How to design MCP tools following prompt-native principles. Tools should be primitives that enable capability, not workflows that encode decisions.

**Core principle:** Whatever a user can do, the agent should be able to do. Don't artificially limit the agent—give it the same primitives a power user would have.
</overview>

<principle name="primitives-not-workflows">
## Tools Are Primitives, Not Workflows

**Wrong approach:** Tools that encode business logic
```typescript
tool("process_feedback", {
  feedback: z.string(),
  category: z.enum(["bug", "feature", "question"]),
  priority: z.enum(["low", "medium", "high"]),
}, async ({ feedback, category, priority }) => {
  // Tool decides how to process
  const processed = categorize(feedback);
  const stored = await saveToDatabase(processed);
  const notification = await notify(priority);
  return { processed, stored, notification };
});
```

**Right approach:** Primitives that enable any workflow
```typescript
tool("store_item", {
  key: z.string(),
  value: z.any(),
}, async ({ key, value }) => {
  await db.set(key, value);
  return { text: `Stored ${key}` };
});

tool("sgai_send_message", {
  channel: z.string(),
  content: z.string(),
}, async ({ channel, content }) => {
  await messenger.send(channel, content);
  return { text: "Sent" };
});
```

The agent decides categorization, priority, and when to notify based on the system prompt.
</principle>

<principle name="descriptive-names">
## Tools Should Have Descriptive, Primitive Names

Names should describe the capability, not the use case:

| Wrong | Right |
|-------|-------|
| `process_user_feedback` | `store_item` |
| `create_feedback_summary` | `write_file` |
| `send_notification` | `sgai_send_message` |
| `deploy_to_production` | `git_push` |

The prompt tells the agent *when* to use primitives. The tool just provides *capability*.
</principle>

<principle name="simple-inputs">
## Inputs Should Be Simple

Tools accept data. They don't accept decisions.

**Wrong:** Tool accepts decisions
```typescript
tool("format_content", {
  content: z.string(),
  format: z.enum(["markdown", "html", "json"]),
  style: z.enum(["formal", "casual", "technical"]),
}, ...)
```

**Right:** Tool accepts data, agent decides format
```typescript
tool("write_file", {
  path: z.string(),
  content: z.string(),
}, ...)
// Agent decides to write index.html with HTML content, or data.json with JSON
```
</principle>

<principle name="rich-outputs">
## Outputs Should Be Rich

Return enough information for the agent to verify and iterate.

**Wrong:** Minimal output
```typescript
async ({ key }) => {
  await db.delete(key);
  return { text: "Deleted" };
}
```

**Right:** Rich output
```typescript
async ({ key }) => {
  const existed = await db.has(key);
  if (!existed) {
    return { text: `Key ${key} did not exist` };
  }
  await db.delete(key);
  return { text: `Deleted ${key}. ${await db.count()} items remaining.` };
}
```
</principle>

<design_template>
## Tool Design Template

```typescript
import { createSdkMcpServer, tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";

export const serverName = createSdkMcpServer({
  name: "server-name",
  version: "1.0.0",
  tools: [
    // READ operations
    tool(
      "read_item",
      "Read an item by key",
      { key: z.string().describe("Item key") },
      async ({ key }) => {
        const item = await storage.get(key);
        return {
          content: [{
            type: "text",
            text: item ? JSON.stringify(item, null, 2) : `Not found: ${key}`,
          }],
          isError: !item,
        };
      }
    ),

    tool(
      "list_items",
      "List all items, optionally filtered",
      {
        prefix: z.string().optional().describe("Filter by key prefix"),
        limit: z.number().default(100).describe("Max items"),
      },
      async ({ prefix, limit }) => {
        const items = await storage.list({ prefix, limit });
        return {
          content: [{
            type: "text",
            text: `Found ${items.length} items:\n${items.map(i => i.key).join("\n")}`,
          }],
        };
      }
    ),

    // WRITE operations
    tool(
      "store_item",
      "Store an item",
      {
        key: z.string().describe("Item key"),
        value: z.any().describe("Item data"),
      },
      async ({ key, value }) => {
        await storage.set(key, value);
        return {
          content: [{ type: "text", text: `Stored ${key}` }],
        };
      }
    ),

    tool(
      "delete_item",
      "Delete an item",
      { key: z.string().describe("Item key") },
      async ({ key }) => {
        const existed = await storage.delete(key);
        return {
          content: [{
            type: "text",
            text: existed ? `Deleted ${key}` : `${key} did not exist`,
          }],
        };
      }
    ),

    // EXTERNAL operations
    tool(
      "call_api",
      "Make an HTTP request",
      {
        url: z.string().url(),
        method: z.enum(["GET", "POST", "PUT", "DELETE"]).default("GET"),
        body: z.any().optional(),
      },
      async ({ url, method, body }) => {
        const response = await fetch(url, { method, body: JSON.stringify(body) });
        const text = await response.text();
        return {
          content: [{
            type: "text",
            text: `${response.status} ${response.statusText}\n\n${text}`,
          }],
          isError: !response.ok,
        };
      }
    ),
  ],
});
```
</design_template>

<example name="feedback-server">
## Example: Feedback Storage Server

This server provides primitives for storing feedback. It does NOT decide how to categorize or organize feedback—that's the agent's job via the prompt.

```typescript
export const feedbackMcpServer = createSdkMcpServer({
  name: "feedback",
  version: "1.0.0",
  tools: [
    tool(
      "store_feedback",
      "Store a feedback item",
      {
        item: z.object({
          id: z.string(),
          author: z.string(),
          content: z.string(),
          importance: z.number().min(1).max(5),
          timestamp: z.string(),
          status: z.string().optional(),
          urls: z.array(z.string()).optional(),
          metadata: z.any().optional(),
        }).describe("Feedback item"),
      },
      async ({ item }) => {
        await db.feedback.insert(item);
        return {
          content: [{
            type: "text",
            text: `Stored feedback ${item.id} from ${item.author}`,
          }],
        };
      }
    ),

    tool(
      "list_feedback",
      "List feedback items",
      {
        limit: z.number().default(50),
        status: z.string().optional(),
      },
      async ({ limit, status }) => {
        const items = await db.feedback.list({ limit, status });
        return {
          content: [{
            type: "text",
            text: JSON.stringify(items, null, 2),
          }],
        };
      }
    ),

    tool(
      "update_feedback",
      "Update a feedback item",
      {
        id: z.string(),
        updates: z.object({
          status: z.string().optional(),
          importance: z.number().optional(),
          metadata: z.any().optional(),
        }),
      },
      async ({ id, updates }) => {
        await db.feedback.update(id, updates);
        return {
          content: [{ type: "text", text: `Updated ${id}` }],
        };
      }
    ),
  ],
});
```

The system prompt then tells the agent *how* to use these primitives:

```markdown
## Feedback Processing

When someone shares feedback:
1. Extract author, content, and any URLs
2. Rate importance 1-5 based on actionability
3. Store using feedback.store_feedback
4. If high importance (4-5), notify the channel

Use your judgment about importance ratings.
```
</example>

<checklist>
## MCP Tool Design Checklist

- [ ] Tool names describe capability, not use case
- [ ] Inputs are data, not decisions
- [ ] Outputs are rich (enough for agent to verify)
- [ ] CRUD operations are separate tools (not one mega-tool)
- [ ] No business logic in tool implementations
- [ ] Error states clearly communicated via `isError`
- [ ] Descriptions explain what the tool does, not when to use it
</checklist>

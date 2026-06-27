import type { Plugin } from "@opencode-ai/plugin"
import { readFile, writeFile } from 'fs/promises';
import { join } from 'path';

export const Workbench: Plugin = async ({ directory }) => {
  const stateFilePath = join(directory, ".sgai", "state.json");
  const knownSessionIDs: Record<string, boolean> = {}
  return {
    config: async (config: any) => {
      config.snapshot = false;
      config.share = "disabled";
      config.autoupdate = false;
    },
    tool: {},
    event: async (input: { event: any; client: any }) => {
      const eventSessionID = sessionIDFromEvent(input?.event);
      if (eventSessionID !== "") {
        const sessionID = eventSessionID;
        const agent = agentNameFromEvent(input.event);
        if (agent !== "" && !knownSessionIDs[sessionID]) {
          knownSessionIDs[sessionID] = true;
          console.log(JSON.stringify({ sessionID, agent }));
        }
      }
      if (input.event.type === "todo.updated") {
        try {
          let currentState: any;
          try {
            const content = await readFile(stateFilePath, 'utf8');
            currentState = JSON.parse(content);
          } catch (error) {
            currentState = {};
          }

          currentState.todos = input.event.properties.todos || [];

          await writeFile(stateFilePath, JSON.stringify(currentState, null, 2));
        } catch (error: any) {
          console.error("Error saving todos: " + error.message);
        }
      }

      if (input.event.type === "session.compacted") {
        try {
          const sessionID = input.event.properties.sessionID;
          await input.client.session.prompt({
            path: { id: sessionID },
            body: {
              parts: [{
                type: "text",
                text: `🔄 **Conversation Compacted**\n\n` +
                  `To maintain context within limits, earlier messages have been summarized.\n\n` +
                  `You MUST re-read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md before continuing.`,
                metadata: {
                  source: "compaction-detector",
                  timestamp: Date.now()
                }
              }]
            }
          });
        } catch (error: any) {
          console.error("Error handling compaction: " + error.message);
        }
      }
    }
  }
}

function agentNameFromEvent(event: any): string {
  switch (event?.type) {
    case "message.updated": {
      const info = event.properties?.info;
      if (info?.role === "user" && typeof info.agent === "string") {
        return cleanAgentName(info.agent);
      }
      return "";
    }
    case "message.part.updated": {
      const part = event.properties?.part;
      if (part?.type === "subtask" && typeof part.agent === "string") {
        return cleanAgentName(part.agent);
      }
      if (part?.type === "agent" && typeof part.name === "string") {
        return cleanAgentName(part.name);
      }
      return "";
    }
    default:
      return "";
  }
}

function sessionIDFromEvent(event: any): string {
  switch (event?.type) {
    case "message.updated":
      return cleanSessionID(event.properties?.info?.sessionID);
    case "message.part.updated":
      return cleanSessionID(event.properties?.part?.sessionID);
    default:
      return cleanSessionID(event?.properties?.sessionID);
  }
}

function cleanAgentName(value: string): string {
  const trimmed = value.trim();
  return trimmed === "unknown" ? "" : trimmed;
}

function cleanSessionID(value: any): string {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

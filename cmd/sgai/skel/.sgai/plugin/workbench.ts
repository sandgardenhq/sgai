import type { Plugin } from "@opencode-ai/plugin"
import { readFile, writeFile } from 'fs/promises';
import { join } from 'path';

export const Workbench: Plugin = async ({ directory }) => {
  const stateFilePath = join(directory, ".sgai", "state.json");

  return {
    config: async (config: any) => {
      config.snapshot = false;
      config.share = "disabled";
      config.autoupdate = false;
      if (!config.instructions) {
        config.instructions = [];
      }
      config.instructions?.unshift(directory + "/.sgai/AGENTS.md");
      config.model = "opencode/big-pickle";

      // Configure MCP server for sgai custom tools (remote HTTP)
      if (!config.mcp) {
        config.mcp = {};
      }
      config.mcp.sgai = {
        type: "remote",
        url: process.env.SGAI_MCP_URL,
        headers: {
          "X-SGAI-Agent-Identity": process.env.SGAI_AGENT_IDENTITY || ""
        },
        timeout: 43200000
      };
    },
    // Tools are now provided by the MCP server configured above
    tool: {},
    event: async (input: { event: any; client: any }) => {
      if (input.event.type === "todo.updated") {
        const currentAgent = process.env.OPENCODE_AGENT_NAME || "unknown";
        if (currentAgent === "coordinator") {
          return;
        }

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
          console.error("\033[1m|\033[0m  \033[0;31mWorkbench\033[0m   Error saving todos: " + error.message + "\033[0m");
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
          console.error("\033[1m|\033[0m  \033[0;31mWorkbench\033[0m   Error handling compaction: " + error.message + "\033[0m");
        }
      }
    },
    "tool.execute.before": async (input: any, output: any) => {
      let currentAgent = "unknown";
      try {
        const content = await readFile(stateFilePath, 'utf8');
        const state = JSON.parse(content);
        currentAgent = state.currentAgent || "unknown";
      } catch (error) {
        // State file doesn't exist or is invalid - use fallback
      }

      const isWriteTool = input.tool === "edit" || input.tool === "write";
      const targetPath = output?.args?.filePath || "";
      const isGoalFile = targetPath.endsWith("GOAL.md") || targetPath.includes("/GOAL.md");

      if (isWriteTool && isGoalFile && currentAgent !== "coordinator") {
        throw new Error(
          `GOAL.md is protected and can only be modified by the coordinator agent.\n\n` +
          `You are currently running as '${currentAgent}'.\n\n` +
          `If you need to request changes to GOAL.md, append the request to .sgai/PROJECT_MANAGEMENT.md and yield to coordinator with workflow navigation.\n\n` +
          `The coordinator will review your request and make the appropriate changes.`
        );
      }
    }
  }
}

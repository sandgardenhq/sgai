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

      // Configure MCP server for sgai custom tools
      if (!config.mcp) {
        config.mcp = {};
      }
      config.mcp.sgai = {
        type: "local",
        command: [process.env.sgai_MCP_EXECUTABLE || "sgai", "mcp"],
        environment: {
          sgai_MCP_WORKING_DIRECTORY: directory
        }
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
          const currentAgent = process.env.OPENCODE_AGENT_NAME || "unknown";

          let currentState: any;
          try {
            const content = await readFile(stateFilePath, 'utf8');
            currentState = JSON.parse(content);
          } catch (error) {
            currentState = { messages: [], messageHistory: [] };
          }

          const pendingMessages = (currentState.messages || []).filter(
            (msg: any) => msg.toAgent === currentAgent
          );

          let inboxPeek = "";
          if (pendingMessages.length > 0) {
            inboxPeek = `\n\n**Inbox Preview (${pendingMessages.length} pending message(s)):**\n`;
            inboxPeek += pendingMessages.map((msg: any) => {
              const subject = msg.body.split('\n')[0];
              return `- From: ${msg.fromAgent} | To: ${msg.toAgent} | Subject: ${subject}`;
            }).join('\n');
            inboxPeek += `\n\nYou MUST call \`check_inbox()\` to read these messages.`;
          }

          await input.client.session.prompt({
            path: { id: sessionID },
            body: {
              parts: [{
                type: "text",
                text: `🔄 **Conversation Compacted**\n\n` +
                      `To maintain context within limits, earlier messages have been summarized.\n\n` +
                      `You MUST re-read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md before continuing.` +
                      inboxPeek,
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
          `If you need to request changes to GOAL.md, use the send_message tool:\n` +
          `  send_message({ toAgent: "coordinator", body: "Please update GOAL.md to ..." })\n\n` +
          `The coordinator will review your request and make the appropriate changes.`
        );
      }
    }
  }
}

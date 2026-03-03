import type { ChatMessage, ChatContext } from "./chat-store";

export interface ChatRequest {
  message: string;
  conversationHistory: { role: "user" | "assistant"; content: string }[];
  context: ChatContext;
}

export interface ChatStreamCallbacks {
  onChunk: (chunk: string) => void;
  onComplete: () => void;
  onError: (error: Error) => void;
}

export async function sendChatMessage(
  request: ChatRequest,
  callbacks: ChatStreamCallbacks,
): Promise<void> {
  const response = await fetch("/api/v1/chat", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => "Unknown error");
    callbacks.onError(new Error(`Chat request failed: ${errorText}`));
    return;
  }

  if (!response.body) {
    callbacks.onError(new Error("No response body"));
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  try {
    let buffer = "";
    while (true) {
      const { done, value } = await reader.read();

      if (done) {
        if (buffer.trim()) {
          processSSELine(buffer, callbacks);
        }
        callbacks.onComplete();
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        processSSELine(line, callbacks);
      }
    }
  } catch (error) {
    callbacks.onError(error instanceof Error ? error : new Error(String(error)));
  }
}

function processSSELine(line: string, callbacks: ChatStreamCallbacks): void {
  const trimmedLine = line.trim();
  if (!trimmedLine || trimmedLine.startsWith(":")) {
    return;
  }

  if (trimmedLine.startsWith("data: ")) {
    const data = trimmedLine.slice(6);
    if (data === "[DONE]") {
      return;
    }
    try {
      const parsed = JSON.parse(data);
      if (parsed.content) {
        callbacks.onChunk(parsed.content);
      } else if (parsed.error) {
        callbacks.onError(new Error(parsed.error));
      }
    } catch {
      callbacks.onChunk(data);
    }
  }
}

export function formatConversationHistory(
  messages: ChatMessage[],
): { role: "user" | "assistant"; content: string }[] {
  return messages
    .filter((msg) => !msg.isStreaming)
    .map((msg) => ({
      role: msg.role,
      content: msg.content,
    }));
}

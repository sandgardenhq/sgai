import { cn } from "@/lib/utils";
import { MarkdownContent } from "../MarkdownContent";
import type { ChatMessage as ChatMessageType } from "@/lib/chat-store";

interface ChatMessageProps {
  message: ChatMessageType;
}

export function ChatMessage({ message }: ChatMessageProps) {
  const isUser = message.role === "user";

  return (
    <div
      className={cn(
        "flex w-full",
        isUser ? "justify-end" : "justify-start"
      )}
    >
      <div
        className={cn(
          "max-w-[85%] rounded-lg px-3 py-2",
          isUser
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-foreground",
          message.isStreaming && "animate-pulse"
        )}
      >
        {isUser ? (
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="text-sm">
            {message.content ? (
              <MarkdownContent content={message.content} />
            ) : (
              <span className="text-muted-foreground italic">Thinking...</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

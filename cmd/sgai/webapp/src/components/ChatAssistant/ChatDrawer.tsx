import { useEffect, useRef, useCallback } from "react";
import { useLocation } from "react-router";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ChatMessage } from "./ChatMessage";
import { ChatInput } from "./ChatInput";
import {
  useChatMessages,
  useChatIsOpen,
  useChatIsStreaming,
  useChatActions,
} from "@/hooks/useChatStore";
import { sendChatMessage, formatConversationHistory } from "@/lib/chat-api";

export function ChatDrawer() {
  const messages = useChatMessages();
  const isOpen = useChatIsOpen();
  const isStreaming = useChatIsStreaming();
  const {
    setOpen,
    addUserMessage,
    startStreaming,
    appendStreamChunk,
    finishStreaming,
    setContext,
  } = useChatActions();

  const scrollRef = useRef<HTMLDivElement>(null);
  const location = useLocation();

  useEffect(() => {
    setContext({
      currentPage: location.pathname,
      workspaceName: extractWorkspaceName(location.pathname),
    });
  }, [location.pathname, setContext]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSendMessage = useCallback(
    async (content: string) => {
      addUserMessage(content);
      startStreaming();

      const conversationHistory = formatConversationHistory(messages);

      await sendChatMessage(
        {
          message: content,
          conversationHistory,
          context: {
            currentPage: location.pathname,
            workspaceName: extractWorkspaceName(location.pathname),
          },
        },
        {
          onChunk: (chunk) => {
            appendStreamChunk(chunk);
          },
          onComplete: () => {
            finishStreaming();
          },
          onError: (error) => {
            console.error("Chat error:", error);
            appendStreamChunk(`\n\nError: ${error.message}`);
            finishStreaming();
          },
        }
      );
    },
    [messages, location.pathname, addUserMessage, startStreaming, appendStreamChunk, finishStreaming]
  );

  const handleOpenChange = useCallback(
    (open: boolean) => {
      setOpen(open);
    },
    [setOpen]
  );

  return (
    <Sheet open={isOpen} onOpenChange={handleOpenChange}>
      <SheetContent
        side="right"
        className="flex flex-col w-full sm:max-w-md p-0"
        showCloseButton={true}
      >
        <SheetHeader className="px-4 pt-4 pb-2 border-b">
          <SheetTitle>SGAI Assistant</SheetTitle>
          <SheetDescription>
            Ask questions about SGAI, your workspace, or get help.
          </SheetDescription>
        </SheetHeader>

        <ScrollArea ref={scrollRef} className="flex-1 p-4">
          <div className="flex flex-col gap-3">
            {messages.length === 0 && (
              <div className="text-center text-muted-foreground text-sm py-8">
                <p>Welcome! I can help you understand SGAI.</p>
                <p className="mt-2">Try asking:</p>
                <ul className="mt-2 space-y-1">
                  <li>&quot;What is SGAI?&quot;</li>
                  <li>&quot;How do I create a workspace?&quot;</li>
                  <li>&quot;What are skills?&quot;</li>
                </ul>
              </div>
            )}
            {messages.map((message) => (
              <ChatMessage key={message.id} message={message} />
            ))}
          </div>
        </ScrollArea>

        <ChatInput onSend={handleSendMessage} disabled={isStreaming} />
      </SheetContent>
    </Sheet>
  );
}

function extractWorkspaceName(pathname: string): string | undefined {
  const match = pathname.match(/^\/workspaces\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

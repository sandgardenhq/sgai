import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MarkdownContent } from "@/components/MarkdownContent";
import { useFactoryState } from "@/lib/factory-state";
import { cn } from "@/lib/utils";
import type { ApiMessageEntry } from "@/types";

interface MessagesTabProps {
  workspaceName: string;
}

function MessagesTabSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 5 }, (_, i) => (
        <Skeleton key={i} className="h-12 w-full rounded" />
      ))}
    </div>
  );
}

function MessageItem({ message }: { message: ApiMessageEntry }) {
  const isUnread = !message.read;
  return (
    <details className="border rounded-lg">
      <summary className={cn(
        "cursor-pointer px-4 py-3 flex items-center gap-2 text-sm flex-nowrap overflow-hidden",
        isUnread && "font-bold"
      )}>
        <span className="whitespace-nowrap shrink-0">{message.fromAgent}</span>
        <span className="whitespace-nowrap shrink-0">{"\u2192"}</span>
        <span className="whitespace-nowrap shrink-0">{message.toAgent}:</span>
        {message.subject && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className={cn(
                "truncate",
                isUnread ? "text-foreground" : "text-muted-foreground"
              )}>{message.subject}</span>
            </TooltipTrigger>
            <TooltipContent className="max-w-sm">{message.subject}</TooltipContent>
          </Tooltip>
        )}
      </summary>
      {message.body && (
        <div className="px-4 pb-4 border-t pt-3 space-y-1">
          <div className="text-xs text-muted-foreground">
            <strong>From:</strong> {message.fromAgent}
          </div>
          <div className="text-xs text-muted-foreground">
            <strong>To:</strong> {message.toAgent}
          </div>
          <MarkdownContent content={message.body} className="mt-2" />
        </div>
      )}
    </details>
  );
}

export function MessagesTab({ workspaceName }: MessagesTabProps) {
  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);

  if (fetchStatus === "fetching" && !workspace) return <MessagesTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load messages
        </p>
      );
    }
    return null;
  }

  const messages = workspace.messages ?? [];

  if (messages.length === 0) {
    return <p className="text-sm italic text-muted-foreground">No messages</p>;
  }

  return (
    <div className="space-y-2">
      {messages.map((msg) => (
        <MessageItem key={msg.id} message={msg} />
      ))}
    </div>
  );
}

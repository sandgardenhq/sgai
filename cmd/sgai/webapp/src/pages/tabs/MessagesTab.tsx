import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { useFactoryState } from "@/lib/factory-state";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Trash2 } from "lucide-react";
import { useTransition, useState, useCallback } from "react";
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

interface MessageItemProps {
  message: ApiMessageEntry;
  workspaceName: string;
  onDelete: (messageId: number) => void;
  isDeleting: boolean;
}

function MessageItem({ message, workspaceName, onDelete, isDeleting }: MessageItemProps) {
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
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="ml-auto h-6 w-6 shrink-0"
          disabled={isDeleting}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onDelete(message.id);
          }}
          aria-label={`Delete message from ${message.fromAgent}`}
        >
          <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
        </Button>
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
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  const handleDelete = useCallback((messageId: number) => {
    setDeleteError(null);
    setDeletingId(messageId);
    startTransition(async () => {
      try {
        const response = await api.workspaces.deleteMessage(workspaceName, messageId);
        if (!response.deleted) {
          setDeleteError(response.message || "Failed to delete message");
        }
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Failed to delete message");
      } finally {
        setDeletingId(null);
      }
    });
  }, [workspaceName]);

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
      {deleteError && (
        <p className="text-sm text-destructive">{deleteError}</p>
      )}
      {messages.map((msg) => (
        <MessageItem 
          key={msg.id} 
          message={msg} 
          workspaceName={workspaceName}
          onDelete={handleDelete}
          isDeleting={deletingId === msg.id || isPending}
        />
      ))}
    </div>
  );
}

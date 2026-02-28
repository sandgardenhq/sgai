import { MessageCircleIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useChatActions, useChatIsOpen } from "@/hooks/useChatStore";
import { cn } from "@/lib/utils";

export function ChatButton() {
  const { toggleOpen } = useChatActions();
  const isOpen = useChatIsOpen();

  return (
    <Button
      onClick={toggleOpen}
      size="icon"
      className={cn(
        "fixed bottom-4 right-4 z-50 size-12 rounded-full shadow-lg",
        "hover:scale-105 transition-transform",
        isOpen && "bg-secondary text-secondary-foreground"
      )}
      aria-label={isOpen ? "Close chat assistant" : "Open chat assistant"}
    >
      <MessageCircleIcon className="size-6" />
    </Button>
  );
}

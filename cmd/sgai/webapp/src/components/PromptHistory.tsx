import { useState } from "react";
import { History, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
  DialogDescription,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface PromptHistoryProps {
  history: string[];
  onSelect: (entry: string) => void;
  onClear: () => void;
  disabled?: boolean;
}

export function PromptHistory({ history, onSelect, onClear, disabled }: PromptHistoryProps) {
  const [open, setOpen] = useState(false);

  if (history.length === 0) return null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <DialogTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={disabled}
              className="gap-1.5"
            >
              <History className="h-3.5 w-3.5" />
              History ({history.length})
            </Button>
          </DialogTrigger>
        </TooltipTrigger>
        <TooltipContent>Browse prompt history</TooltipContent>
      </Tooltip>

      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Prompt History</DialogTitle>
          <DialogDescription>
            Select a previous prompt to re-fill the input field.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[300px]">
          <ul className="space-y-2">
            {history.map((entry, idx) => (
              <li key={`${idx}-${entry.slice(0, 20)}`}>
                <DialogClose asChild>
                  <button
                    type="button"
                    className="w-full text-left rounded-md border p-3 text-sm hover:bg-muted/50 transition-colors cursor-pointer"
                    onClick={() => {
                      onSelect(entry);
                      setOpen(false);
                    }}
                  >
                    <span className="line-clamp-2 break-words">{entry}</span>
                  </button>
                </DialogClose>
              </li>
            ))}
          </ul>
        </ScrollArea>

        <div className="flex justify-end pt-2">
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={() => {
              onClear();
              setOpen(false);
            }}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Clear History
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

import { Separator } from "@/components/ui/separator";
import { MarkdownContent } from "@/components/MarkdownContent";

interface ResponseContextProps {
  goalContent?: string;
  projectMgmtContent?: string;
}

export function ResponseContext({ goalContent, projectMgmtContent }: ResponseContextProps) {
  if (!goalContent && !projectMgmtContent) {
    return null;
  }

  return (
    <>
      <Separator className="my-4" />
      <div className="space-y-3">
        {goalContent && (
          <details className="group border rounded-lg overflow-hidden">
            <summary className="flex items-center gap-2 px-4 py-3 cursor-pointer bg-muted/30 text-sm font-medium select-none hover:bg-muted/50 transition-colors">
              <span className="text-[0.7rem] transition-transform group-open:rotate-90">&#x25B6;</span>
              <span className="flex-1">GOAL.md</span>
            </summary>
            <MarkdownContent
              content={goalContent}
              className="px-6 py-4 text-sm leading-relaxed"
            />
          </details>
        )}

        {projectMgmtContent && (
          <details className="group border rounded-lg overflow-hidden">
            <summary className="flex items-center gap-2 px-4 py-3 cursor-pointer bg-muted/30 text-sm font-medium select-none hover:bg-muted/50 transition-colors">
              <span className="text-[0.7rem] transition-transform group-open:rotate-90">&#x25B6;</span>
              <span className="flex-1">PROJECT_MANAGEMENT.md</span>
            </summary>
            <MarkdownContent
              content={projectMgmtContent}
              className="px-6 py-4 text-sm leading-relaxed"
            />
          </details>
        )}
      </div>
    </>
  );
}

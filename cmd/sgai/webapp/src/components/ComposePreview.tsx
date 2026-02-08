import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AlertDescription, Alert } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { FileText, AlertTriangle } from "lucide-react";
import type { ApiComposePreviewResponse } from "@/types";

interface ComposePreviewProps {
  preview: ApiComposePreviewResponse | null;
  isLoading?: boolean;
  title?: string;
}

export function ComposePreview({
  preview,
  isLoading = false,
  title = "GOAL.md Preview",
}: ComposePreviewProps) {
  return (
    <Card className="sticky top-0 max-h-[calc(100vh-10rem)] overflow-hidden flex flex-col">
      <CardHeader className="border-b py-3 px-4 flex-shrink-0">
        <CardTitle className="text-sm flex items-center gap-2">
          <FileText className="h-4 w-4" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-y-auto p-4">
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-2/3" />
          </div>
        ) : (
          <>
            <pre className="text-xs leading-relaxed whitespace-pre-wrap break-words font-mono text-muted-foreground">
              {preview?.content ?? "No preview available"}
            </pre>
            {preview?.flowError ? (
              <Alert className="mt-3 border-destructive/50 text-destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{preview.flowError}</AlertDescription>
              </Alert>
            ) : null}
          </>
        )}
      </CardContent>
    </Card>
  );
}

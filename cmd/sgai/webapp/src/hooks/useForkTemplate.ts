import { useState, useEffect } from "react";
import { api } from "@/lib/api";

export function useForkTemplate(workspaceName: string): string {
  const [content, setContent] = useState("");

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;

    api.workspaces.forkTemplate(workspaceName).then(
      (result) => {
        if (!cancelled && result.content) {
          setContent(result.content);
        }
      },
      () => {},
    );

    return () => {
      cancelled = true;
    };
  }, [workspaceName]);

  return content;
}

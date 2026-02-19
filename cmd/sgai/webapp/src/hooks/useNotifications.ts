import { useEffect, useRef } from "react";
import { useSSEEvent } from "./useSSE";
import { api } from "../lib/api";
import type { ApiWorkspaceEntry } from "../types";

function collectNeedsInput(
  workspaces: ApiWorkspaceEntry[],
  out: Map<string, boolean>,
): void {
  for (const ws of workspaces) {
    out.set(ws.name, ws.needsInput);
    if (ws.forks) {
      collectNeedsInput(ws.forks, out);
    }
  }
}

function fireNotification(workspaceName: string): void {
  if (!("Notification" in window)) {
    return;
  }

  if (Notification.permission !== "granted") {
    return;
  }

  const notification = new Notification("Approval Needed", {
    body: `Workspace ${workspaceName} needs your input`,
    tag: workspaceName,
  });

  notification.onclick = () => {
    window.focus();
  };
}

export function useNotifications(): void {
  const event = useSSEEvent("workspace:update");
  const previousStateRef = useRef<Map<string, boolean>>(new Map());

  useEffect(() => {
    if (event === null) {
      return;
    }

    let cancelled = false;

    api.workspaces.list().then((response) => {
      if (cancelled) {
        return;
      }

      const currentState = new Map<string, boolean>();
      collectNeedsInput(response.workspaces, currentState);

      const previous = previousStateRef.current;

      for (const [name, needsInput] of currentState) {
        const wasNeedingInput = previous.get(name) ?? false;
        if (!wasNeedingInput && needsInput) {
          fireNotification(name);
        }
      }

      previousStateRef.current = currentState;
    }).catch((err: unknown) => {
      console.error("useNotifications: failed to fetch workspaces:", err);
    });

    return () => {
      cancelled = true;
    };
  }, [event]);
}

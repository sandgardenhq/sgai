import { useEffect, useRef } from "react";
import { useFactoryState } from "../lib/factory-state";
import type { ApiWorkspaceEntry } from "../lib/factory-state";

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
  const { workspaces, lastFetchedAt } = useFactoryState();
  const previousStateRef = useRef<Map<string, boolean>>(new Map());

  useEffect(() => {
    if (lastFetchedAt === null) {
      return;
    }

    const currentState = new Map<string, boolean>();
    collectNeedsInput(workspaces, currentState);

    const previous = previousStateRef.current;

    for (const [name, needsInput] of currentState) {
      const wasNeedingInput = previous.get(name) ?? false;
      if (!wasNeedingInput && needsInput) {
        fireNotification(name);
      }
    }

    previousStateRef.current = currentState;
  }, [workspaces, lastFetchedAt]);
}

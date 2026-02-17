import { useState, useCallback } from "react";
import { Button } from "./ui/button";

const DISMISSED_KEY = "notification-permission-dismissed";

function isDismissed(): boolean {
  try {
    return localStorage.getItem(DISMISSED_KEY) === "true";
  } catch {
    return false;
  }
}

function persistDismissal(): void {
  try {
    localStorage.setItem(DISMISSED_KEY, "true");
  } catch {
    // localStorage unavailable
  }
}

function shouldShowBar(): boolean {
  if (!("Notification" in window)) {
    return false;
  }
  if (Notification.permission !== "default") {
    return false;
  }
  return !isDismissed();
}

export function NotificationPermissionBar() {
  const [visible, setVisible] = useState(shouldShowBar);

  const handleEnable = useCallback(async () => {
    if (!("Notification" in window)) {
      return;
    }
    const result = await Notification.requestPermission();
    if (result === "granted" || result === "denied") {
      setVisible(false);
    }
  }, []);

  const handleDismiss = useCallback(() => {
    persistDismissal();
    setVisible(false);
  }, []);

  if (!visible) {
    return null;
  }

  return (
    <div
      role="alert"
      aria-live="polite"
      className="fixed top-0 left-0 right-0 z-50 bg-yellow-500 text-yellow-950 py-2 px-4 text-sm font-medium flex items-center justify-between gap-2"
    >
      <span>Enable browser notifications to get alerted when a workspace needs your input.</span>
      <div className="flex items-center gap-2 shrink-0">
        <Button
          size="sm"
          variant="outline"
          className="bg-yellow-600 text-white border-yellow-700 hover:bg-yellow-700 hover:text-white h-7 text-xs"
          onClick={handleEnable}
        >
          Enable
        </Button>
        <Button
          size="sm"
          variant="ghost"
          className="text-yellow-950 hover:bg-yellow-600 hover:text-yellow-950 h-7 text-xs"
          onClick={handleDismiss}
        >
          Dismiss
        </Button>
      </div>
    </div>
  );
}

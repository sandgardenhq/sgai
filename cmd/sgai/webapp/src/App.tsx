import { Outlet } from "react-router";
import { AppStateProvider } from "./contexts/AppStateProvider";
import { ConnectionStatusBanner } from "./components/ConnectionStatusBanner";
import { NotificationPermissionBar } from "./components/NotificationPermissionBar";
import { ChatAssistant } from "./components/ChatAssistant";
import { TooltipProvider } from "./components/ui/tooltip";
import { useNotifications } from "./hooks/useNotifications";

export function App() {
  useNotifications();

  return (
    <AppStateProvider>
      <TooltipProvider>
        <NotificationPermissionBar />
        <ConnectionStatusBanner />
        <div className="min-h-screen bg-background text-foreground">
          <main className="p-4">
            <Outlet />
          </main>
        </div>
        <ChatAssistant />
      </TooltipProvider>
    </AppStateProvider>
  );
}

import { Outlet } from "react-router";
import { AppStateProvider } from "./contexts/AppStateProvider";
import { ConnectionStatusBanner } from "./components/ConnectionStatusBanner";
import { TooltipProvider } from "./components/ui/tooltip";

export function App() {
  return (
    <AppStateProvider>
      <TooltipProvider>
        <ConnectionStatusBanner />
        <div className="min-h-screen bg-background text-foreground">
          <main className="p-4">
            <Outlet />
          </main>
        </div>
      </TooltipProvider>
    </AppStateProvider>
  );
}

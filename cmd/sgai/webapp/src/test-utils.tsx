import { ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";

interface TestWrapperProps {
  children: ReactNode;
  initialRoute?: string;
}

export function TestWrapper({ children, initialRoute = "/" }: TestWrapperProps) {
  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      <TooltipProvider>
        <SidebarProvider>
          {children}
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>
  );
}

interface RouteWrapperProps {
  children: ReactNode;
  path: string;
  element: ReactNode;
}

export function RouteWrapper({ children, path, element }: RouteWrapperProps) {
  return (
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <SidebarProvider>
          <Routes>
            <Route path={path} element={element} />
          </Routes>
          {children}
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>
  );
}

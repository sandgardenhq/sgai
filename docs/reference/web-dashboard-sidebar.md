# Web dashboard sidebar layout

This page shows how the web dashboard structures its navigation sidebar using the shared Sidebar primitives.

The sidebar uses a Sheet-style overlay on small screens. This keeps the mobile sidebar content in a portal-based layer instead of inside the normal page flow.

## What you build

- A sidebar wrapped in `SidebarProvider` so the open/collapsed state is shared.
- A mobile header with a `SidebarTrigger`.
- A navigation list built from `SidebarMenu`, `SidebarMenuItem`, and `SidebarMenuButton`.

## Prerequisites

- A React page layout that can render a sidebar and a main content area.
- Access to the shared sidebar primitives exported from `@/components/ui/sidebar`.

## Basic layout

1. Wrap the page in `SidebarProvider`.
2. Render a `Sidebar` for navigation.
3. Render your main content area as a sibling of the sidebar.

```tsx
import { ReactNode } from "react"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"

type Props = {
  children: ReactNode
}

export function PageLayout({ children }: Props) {
  return (
    <SidebarProvider>
      <div className="flex min-h-[calc(100vh-4rem)] w-full">
        <Sidebar side="left" collapsible="offcanvas">
          <SidebarHeader className="p-4">
            <div className="text-sm font-semibold">Workspaces</div>
          </SidebarHeader>

          <SidebarContent>
            {/* Put navigation here */}
          </SidebarContent>

          <SidebarFooter className="p-4">
            {/* Put footer content here */}
          </SidebarFooter>
        </Sidebar>

        <div className="flex-1 flex flex-col min-w-0">
          <div className="md:hidden flex items-center gap-3 px-4 py-3">
            <SidebarTrigger />
            <span className="text-sm font-semibold">Workspaces</span>
          </div>

          <main className="flex-1 overflow-auto">{children}</main>
        </div>
      </div>
    </SidebarProvider>
  )
}
```

At this point, you should have a sidebar region (desktop) and a trigger button visible on smaller screens.

## Build a menu

Use `SidebarMenu` + `SidebarMenuItem` + `SidebarMenuButton` to build a navigation list.

The dashboard uses `SidebarMenuButton` with:

- `asChild` to render an existing link component.
- `isActive` to reflect the currently-selected route/item.

```tsx
import { Link } from "react-router"

import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

type MenuItem = {
  to: string
  label: string
  isActive: boolean
}

export function WorkspaceMenu({ items }: { items: MenuItem[] }) {
  return (
    <SidebarMenu>
      {items.map((item) => (
        <SidebarMenuItem key={item.to}>
          <SidebarMenuButton asChild isActive={item.isActive}>
            <Link to={item.to}>{item.label}</Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}
```

## Control mobile open/close state

The dashboard closes the mobile sidebar when the selected workspace changes by calling `setOpenMobile(false)` from `useSidebar()`.

Use this pattern when selecting an item should dismiss the mobile navigation.

```tsx
import { useEffect } from "react"

import { useSidebar } from "@/components/ui/sidebar"

export function CloseSidebarOnSelection({ selectionKey }: { selectionKey: string }) {
  const { setOpenMobile } = useSidebar()

  useEffect(() => {
    setOpenMobile(false)
  }, [selectionKey, setOpenMobile])

  return null
}
```

## Troubleshooting

### “useSidebar must be used within a SidebarProvider”

Wrap the part of the React tree that calls `useSidebar()` in `SidebarProvider`.

### Mobile overlay styles look wrong

The sidebar uses a Sheet-based overlay on mobile viewports.

1. Check the global stylesheet for the sidebar-related CSS custom properties and base layer styles.
2. Verify that the sidebar content renders inside the overlay on small screens (the overlay should cover the app content, instead of pushing it aside).

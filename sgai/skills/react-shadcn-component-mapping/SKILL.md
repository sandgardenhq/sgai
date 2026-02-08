---
name: react-shadcn-component-mapping
description: Maps PicoCSS patterns to shadcn/ui equivalents for all 44 HTMX templates. Use when converting HTMX templates to React components, choosing shadcn/ui components for a page, or reviewing React component choices against PicoCSS originals. Triggers on template conversion, component selection, PicoCSS-to-React migration tasks.
---

# React shadcn/ui Component Mapping

## Overview

Lookup table mapping every PicoCSS element and pattern used in the 44 HTMX templates to its shadcn/ui equivalent. Use this when converting templates to React components to ensure consistent, accessible replacements.

**Reference:** [shadcn/ui docs](https://ui.shadcn.com/docs)

## When to Use

- Use when converting an HTMX template to a React page component
- Use when choosing which shadcn/ui component to use for a UI pattern
- Use when reviewing React components for correct shadcn/ui usage
- Don't use for custom components that have no PicoCSS/shadcn equivalent

## Element Mapping Table

### Layout & Structure

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `<main class="container">` | Layout component + `className="container mx-auto"` | Tailwind utility | Use Tailwind container |
| `<nav>` (sidebar) | `<Sidebar>` | `@/components/ui/sidebar` | shadcn Sidebar component |
| `<nav>` (top navigation) | `<NavigationMenu>` | `@/components/ui/navigation-menu` | Or custom header |
| `<article>` | `<Card>` | `@/components/ui/card` | Card with CardHeader, CardContent |
| `<section>` | `<Card>` or plain `<div>` | - | Depends on visual treatment |
| `role="group"` (button groups) | `<div className="flex gap-2">` | Tailwind utility | Flex container |
| Resizable panels | `<ResizablePanel>` | `@/components/ui/resizable` | For sidebar + content layout |

### Data Display

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `<table>` | `<Table>` | `@/components/ui/table` | With TableHeader, TableBody, TableRow, TableCell |
| `<details>` / `<summary>` | `<Accordion>` | `@/components/ui/accordion` | Or `<Collapsible>` for single items |
| `<details>` (single toggle) | `<Collapsible>` | `@/components/ui/collapsible` | Simpler than Accordion |
| `<mark>` / highlights | `<Badge>` | `@/components/ui/badge` | Status indicators, tags |
| `<code>` / `<pre>` | `<code>` + Tailwind | - | Keep native with styling |
| Data tooltip (`data-tooltip`) | `<Tooltip>` | `@/components/ui/tooltip` | Wrap with TooltipProvider, TooltipTrigger, TooltipContent |
| Overflow with ellipsis | `<Tooltip>` wrapping truncated text | `@/components/ui/tooltip` | `className="truncate"` + tooltip for full text |
| `<progress>` | `<Progress>` | `@/components/ui/progress` | Progress bar |
| Scrollable content | `<ScrollArea>` | `@/components/ui/scroll-area` | Virtualized scrolling |

### Interactive Elements

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `<button>` | `<Button>` | `@/components/ui/button` | Variants: default, destructive, outline, secondary, ghost, link |
| `<a>` styled as button | `<Button asChild>` wrapping `<Link>` | `@/components/ui/button` | Use React Router `<Link>` inside |
| `<button class="secondary">` | `<Button variant="secondary">` | - | Map PicoCSS classes to variants |
| `<button class="outline">` | `<Button variant="outline">` | - | |
| `aria-busy="true"` on button | `<Button disabled>` + spinner | - | Loading state |
| Tab navigation (custom) | `<Tabs>` | `@/components/ui/tabs` | TabsList, TabsTrigger, TabsContent |
| `<dialog>` | `<Dialog>` | `@/components/ui/dialog` | DialogTrigger, DialogContent, DialogHeader, etc. |
| `<dialog>` (confirmation) | `<AlertDialog>` | `@/components/ui/alert-dialog` | For destructive/confirmation actions |
| Dropdown menu | `<DropdownMenu>` | `@/components/ui/dropdown-menu` | Trigger, Content, Item, etc. |

### Forms

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `<input type="text">` | `<Input>` | `@/components/ui/input` | |
| `<textarea>` | `<Textarea>` | `@/components/ui/textarea` | |
| `<select>` | `<Select>` | `@/components/ui/select` | SelectTrigger, SelectContent, SelectItem |
| `<input type="checkbox">` | `<Checkbox>` | `@/components/ui/checkbox` | |
| `<input type="radio">` | `<RadioGroup>` | `@/components/ui/radio-group` | RadioGroupItem |
| `<label>` | `<Label>` | `@/components/ui/label` | |
| `<fieldset>` | `<div className="space-y-4">` | Tailwind | Group form fields |
| Form validation errors | `<p className="text-sm text-destructive">` | Tailwind | Below the input |
| `<input type="search">` | `<Input>` with search icon | `@/components/ui/input` | Add `<Search>` icon via lucide-react |

### Feedback & Status

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `role="alert"` | `<Alert>` | `@/components/ui/alert` | AlertTitle, AlertDescription |
| `aria-busy="true"` (loading) | `<Skeleton>` | `@/components/ui/skeleton` | Multiple skeleton lines |
| Loading spinners | `<Skeleton>` or custom spinner | `@/components/ui/skeleton` | Prefer Skeleton for content areas |
| Toast/notification | `<Sonner>` (toast) | `@/components/ui/sonner` | Via `toast()` function |
| Empty state messages | Custom component | - | Card with icon + message |
| Error boundaries | Custom ErrorBoundary | - | Wrap with React error boundary |

### Navigation & Routing

| PicoCSS / HTML Pattern | shadcn/ui Component | Import | Notes |
|------------------------|--------------------|---------|----|
| `<a href="...">` (internal) | `<Link>` from React Router | `react-router` | NOT `<a>` tag |
| Breadcrumbs | `<Breadcrumb>` | `@/components/ui/breadcrumb` | BreadcrumbList, BreadcrumbItem, etc. |
| Pagination | `<Pagination>` | `@/components/ui/pagination` | If needed for lists |

## Template-by-Template Mapping

### M1: Entity Browsers

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| `agents.html` | `AgentList` | Card, Table, Badge, ScrollArea |
| `skills.html` | `SkillList` | Card, Table, Badge |
| `skill_detail.html` | `SkillDetail` | Card, Tabs, ScrollArea, Badge |
| `snippets.html` | `SnippetList` | Card, Table, Tabs (language filter) |
| `snippet_detail.html` | `SnippetDetail` | Card, ScrollArea, Badge |

### M2: Dashboard + Workspace Tree

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| `trees.html` | `Dashboard` | Sidebar, ResizablePanel |
| `trees_content.html` | `Dashboard` (content area) | Tabs, Card |
| `trees_workspace.html` | `WorkspaceDetail` | Card, Badge, Tabs |
| `trees_root_workspace.html` | `Dashboard` (root) | Sidebar, Button |
| `trees_no_workspace.html` | `EmptyState` | Card (empty state pattern) |

### M3: Session Tabs

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| `trees_session_content.html` | `SessionTab` | Card, Badge, Button, Tooltip |
| `trees_specification_content.html` | `SpecificationTab` | Card, ScrollArea |
| `trees_messages_content.html` | `MessagesTab` | Table, Badge, ScrollArea |
| `trees_log_content.html` | `LogTab` | ScrollArea, code block |
| `trees_run_content.html` | `RunTab` | Card, Badge, Button |
| `trees_changes_content.html` | `ChangesTab` | ScrollArea, code block |
| `trees_events_content.html` | `EventsTab` | Table, Badge, ScrollArea |
| `trees_forks_content.html` | `ForksTab` | Table, Badge, Button |
| `trees_retrospectives_content.html` | `RetrospectivesTab` | Card, Accordion, Button |
| `trees_retrospectives_apply_select.html` | `RetrospectivesTab` | Select, Button, Dialog |

### M4: Response System

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| `response_multichoice.html` | `ResponseMultiChoice` | RadioGroup, Button, Card |
| `response_multichoice_modal.html` | `ResponseModal` | Dialog, RadioGroup, Button |
| `response_context.html` | `ResponseContext` | Textarea, Button, Alert |
| `trees_actions.html` | `SessionControls` | Button, DropdownMenu |
| `trees_root_actions.html` | `SessionControls` | Button |
| `trees_reset_banner.html` | `ResetBanner` | Alert, Button |

### M5: GOAL Composer Wizard

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| `compose_landing.html` | `ComposeLanding` | Card, Button |
| `compose_wizard_base.html` | `WizardLayout` | Stepper (custom), Progress |
| `compose_wizard_step1.html` | `WizardStep1` | Form, Select, Input |
| `compose_wizard_step2.html` | `WizardStep2` | Form, Textarea, Select |
| `compose_wizard_step3.html` | `WizardStep3` | Form, Textarea, Checkbox |
| `compose_wizard_step4.html` | `WizardStep4` | Form, Textarea |
| `compose_wizard_finish.html` | `WizardFinish` | Card, Button, Alert |
| `compose_preview.html` | `ComposePreview` | Card, ScrollArea, code block |
| `compose_preview_partial.html` | `ComposePreview` | ScrollArea |

### M6: Workspace Management + Remaining

| Template | React Component | Key shadcn Components |
|----------|----------------|----------------------|
| Remaining ~8 templates | Various workspace mgmt pages | Dialog, Form, Input, Button, Select, Alert |

## Common Pattern Conversions

### Loading States

**PicoCSS:**
```html
<button aria-busy="true">Loading...</button>
<article aria-busy="true">Loading...</article>
```

**shadcn/ui:**
```tsx
<Button disabled>
  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
  Loading...
</Button>

<Card>
  <CardContent>
    <Skeleton className="h-4 w-full" />
    <Skeleton className="h-4 w-3/4" />
  </CardContent>
</Card>
```

### Tooltips (Overflow Ellipsis Pattern)

**PicoCSS:**
```html
<span data-tooltip="Full text here">Truncated te...</span>
```

**shadcn/ui:**
```tsx
<TooltipProvider>
  <Tooltip>
    <TooltipTrigger asChild>
      <span className="truncate max-w-[200px] block">Full text here</span>
    </TooltipTrigger>
    <TooltipContent>
      <p>Full text here</p>
    </TooltipContent>
  </Tooltip>
</TooltipProvider>
```

### Tab Navigation

**PicoCSS / HTMX:**
```html
<nav>
  <a href="#session" class="active">Session</a>
  <a href="#log">Log</a>
  <a href="#messages">Messages</a>
</nav>
```

**shadcn/ui:**
```tsx
<Tabs defaultValue="session">
  <TabsList>
    <TabsTrigger value="session">Session</TabsTrigger>
    <TabsTrigger value="log">Log</TabsTrigger>
    <TabsTrigger value="messages">Messages</TabsTrigger>
  </TabsList>
  <TabsContent value="session"><SessionTab /></TabsContent>
  <TabsContent value="log"><LogTab /></TabsContent>
  <TabsContent value="messages"><MessagesTab /></TabsContent>
</Tabs>
```

### Confirmation Dialogs

**PicoCSS / HTMX:**
```html
<dialog id="confirm">
  <article>
    <h3>Are you sure?</h3>
    <footer>
      <button class="secondary" onclick="this.closest('dialog').close()">Cancel</button>
      <button class="contrast" hx-post="/action">Confirm</button>
    </footer>
  </article>
</dialog>
```

**shadcn/ui:**
```tsx
<AlertDialog>
  <AlertDialogTrigger asChild>
    <Button variant="destructive">Delete</Button>
  </AlertDialogTrigger>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>Are you sure?</AlertDialogTitle>
      <AlertDialogDescription>This action cannot be undone.</AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>Cancel</AlertDialogCancel>
      <AlertDialogAction onClick={handleConfirm}>Confirm</AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

## Rules

1. **Always prefer shadcn/ui over custom components** — If a shadcn component exists for the pattern, use it. Don't create custom implementations.

2. **Preserve accessibility** — PicoCSS has good defaults. shadcn/ui also has good accessibility. Ensure ARIA attributes from HTMX templates are preserved or improved.

3. **Use Tooltip for overflow ellipsis** — When content is truncated with `className="truncate"`, always wrap in a `<Tooltip>` showing the full text.

4. **Map PicoCSS button classes to shadcn variants** — `class="secondary"` → `variant="secondary"`, `class="outline"` → `variant="outline"`, `class="contrast"` → `variant="destructive"` (for destructive actions).

5. **Use React Router `<Link>` for navigation** — Never use raw `<a>` tags for internal links. Use `<Link>` from React Router, or `<Button asChild>` wrapping `<Link>`.

## Checklist

Before completing a template conversion, verify:

- [ ] All PicoCSS elements mapped to shadcn/ui equivalents
- [ ] No custom components where shadcn equivalent exists
- [ ] Tooltips added for truncated/overflow text
- [ ] Loading states use `<Skeleton>` not raw `aria-busy`
- [ ] Dialogs use `<Dialog>` or `<AlertDialog>` (not custom modals)
- [ ] Internal links use React Router `<Link>`
- [ ] Forms use shadcn Form, Input, Select, Textarea, etc.
- [ ] Accessibility attributes preserved from HTMX originals

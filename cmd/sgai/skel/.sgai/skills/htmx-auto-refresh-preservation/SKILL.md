---
name: htmx-auto-refresh-preservation
description: Use when building any HTMX interface with polling, SSE, or auto-refresh - prevents state loss (form inputs, scroll positions, details/accordion state) during automatic DOM updates by enforcing Idiomorph morph swaps with correct configuration. When building HTMX pages that auto-refresh via polling (hx-trigger="every Ns") or SSE. When select boxes close on refresh. When form inputs lose focus or reset. When details/accordion elements collapse. When scroll positions jump to top. When you see innerHTML swap on auto-refreshing content.
metadata:
  languages: [html, javascript]
  dependencies: [htmx, idiomorph]
---

# HTMX Auto-Refresh State Preservation

## Overview

Auto-refreshing HTMX interfaces (polling, SSE) destroy user state on every update cycle unless Idiomorph morph swaps are used with correct configuration. This skill captures the mandatory setup and patterns to prevent state loss.

**Core principle:** Every auto-refreshing region MUST use morph swaps with Idiomorph defaults configured to preserve active elements, details state, and scroll positions.

## When to Use

- Use when building any HTMX page with `hx-trigger="every Ns"` polling
- Use when building HTMX pages with SSE-driven content updates
- Use when any region of the page refreshes automatically while user may be interacting
- Use when you see select boxes closing, inputs losing focus, accordions collapsing, or scroll jumping during auto-refresh

Do NOT use for:
- Static pages with no auto-refresh
- One-shot HTMX requests (click-triggered, no polling)
- Pages where full page reload is acceptable

## Quick Reference

| Problem | Solution | Layer |
|---------|----------|-------|
| Full DOM rebuild on refresh | Use `morph:innerHTML` swap | Layer 2 |
| Select box closes on refresh | `ignoreActive = true` | Layer 1 |
| Input loses value on refresh | `ignoreActiveValue = true` | Layer 1 |
| `<details>` collapses on refresh | `beforeAttributeUpdated` callback | Layer 3b |
| Scroll jumps to top on refresh | `htmx:beforeSwap`/`afterSwap` handlers | Layer 3c |
| Form interaction interrupted | `beforeNodeMorphed` exclusion or conditional polling | Layer 3a / 4b |
| Including wrong CDN files | Use ONLY `idiomorph-ext.min.js` | Layer 1 |

---

## Layer 1: Setup (MANDATORY)

Every auto-refreshing HTMX page MUST include this setup. No exceptions.

### CDN Import

MUST use ONLY `idiomorph-ext.min.js` - this single file includes BOTH the Idiomorph core AND the HTMX extension:

```html
<script src="https://cdn.jsdelivr.net/npm/idiomorph@0.3.0/dist/idiomorph-ext.min.js"></script>
```

**ANTI-PATTERN: Including BOTH `idiomorph.min.js` AND `idiomorph-ext.min.js`** - the ext file already includes everything. Adding both causes duplicate registration and unpredictable behavior.

**ANTI-PATTERN: Including only `idiomorph.min.js` without the ext** - this provides the morphing engine but NOT the HTMX extension integration. Morph swaps will not work.

### Body Extension

MUST add `hx-ext="morph"` on the `<body>` element:

```html
<body hx-ext="morph">
```

### Idiomorph Defaults

MUST set these defaults in a `<script>` block after the body opens (or at end of body):

```javascript
Idiomorph.defaults.ignoreActive = true;
Idiomorph.defaults.ignoreActiveValue = true;
```

- `ignoreActive = true`: Prevents morphing the currently focused element (select boxes stay open, inputs keep focus)
- `ignoreActiveValue = true`: Prevents updating the value of the active element (user's typed text is preserved)

**ANTI-PATTERN: Not setting `ignoreActive`** - causes select boxes to snap closed and inputs to lose focus on every refresh cycle. This is the #1 source of "auto-refresh breaks my form" bugs.

---

## Layer 2: Swap Strategies

### Container Content Refresh

MUST use `hx-swap="morph:innerHTML"` when refreshing the contents of a container (keeps the container element, morphs its children):

```html
<div hx-get="/refresh-endpoint"
     hx-trigger="every 2s"
     hx-swap="morph:innerHTML">
  <!-- content refreshed via morph -->
</div>
```

### Full Element Replacement

Use `hx-swap="morph:outerHTML"` or `hx-swap="morph"` when replacing an entire element including itself:

```html
<div hx-get="/replace-self"
     hx-trigger="every 5s"
     hx-swap="morph:outerHTML">
</div>
```

### Rules

- MUST use `morph:innerHTML` or `morph:outerHTML` on any auto-refreshing region
- MUST NOT use plain `innerHTML` or `outerHTML` swap on auto-refreshing content

**ANTI-PATTERN: Using `hx-swap="innerHTML"` instead of `hx-swap="morph:innerHTML"`** - causes full DOM replacement on every cycle, destroying all element state (focus, scroll, open/closed, values).

---

## Layer 3: State Preservation

### 3a. Form/Input Preservation

The `ignoreActive=true` and `ignoreActiveValue=true` defaults from Layer 1 handle most form cases automatically. The active (focused) element and its value are left untouched during morph.

For complex form containers that must be completely excluded from morphing (e.g., a prompt input area with multiple interactive elements), use `beforeNodeMorphed`:

```javascript
Idiomorph.defaults.callbacks.beforeNodeMorphed = function(oldNode, newNode) {
  if (oldNode.id === 'my-form-container') return false;
  return true;
};
```

Returning `false` from `beforeNodeMorphed` skips morphing that node and all its children entirely.

**ANTI-PATTERN: Not setting `ignoreActive`/`ignoreActiveValue` and relying solely on `beforeNodeMorphed`** - this is the nuclear option. Use it only for containers that need complete exclusion. The defaults handle individual active elements.

### 3b. Details/Accordion State

Idiomorph morphing updates attributes, which resets the `open` attribute on `<details>` elements (closing them). Use `beforeAttributeUpdated` to preserve:

```javascript
Idiomorph.defaults.callbacks.beforeAttributeUpdated = function(attr, node) {
  if (attr === 'open' && node.tagName === 'DETAILS') return false;
};
```

Returning `false` from `beforeAttributeUpdated` prevents that specific attribute update.

**ANTI-PATTERN: `<details>` elements collapsing to closed state on every morph refresh** - users cannot keep sections expanded. This callback is mandatory when auto-refreshing pages contain `<details>` elements.

### 3c. Scroll Position

Idiomorph does NOT automatically preserve scroll positions. Scrollable panels jump to the top after each morph. Use HTMX events to save/restore:

```javascript
(function() {
  var scrollState = {}, selectors = '.scrollable-panel-1, .scrollable-panel-2';
  document.body.addEventListener('htmx:beforeSwap', function() {
    scrollState = {};
    document.querySelectorAll(selectors).forEach(function(el, i) {
      scrollState[i] = el.scrollTop;
    });
  });
  document.body.addEventListener('htmx:afterSwap', function() {
    document.querySelectorAll(selectors).forEach(function(el, i) {
      if (scrollState[i] !== undefined) el.scrollTop = scrollState[i];
    });
  });
})();
```

Update the `selectors` variable to match your scrollable containers.

**ANTI-PATTERN: Scrollable panels jumping to the top on every refresh cycle** - users lose their reading position. Any scrollable container within an auto-refreshing region needs this handler.

---

## Layer 4: Advanced Patterns

### 4a. Selective Morph Exclusion

Use `beforeNodeMorphed` returning `false` for specific element IDs to completely exclude them from morphing:

```javascript
Idiomorph.defaults.callbacks.beforeNodeMorphed = function(oldNode, newNode) {
  if (oldNode.id === 'adhoc-prompt-container') return false;
  if (oldNode.id === 'chat-input-area') return false;
  return true;
};
```

This is the nuclear option for elements that must NEVER be touched during refresh (e.g., active prompt containers, rich editors, drag-and-drop zones).

**When to use selective exclusion vs `ignoreActive`:**
- `ignoreActive`: Protects whichever single element currently has focus
- `beforeNodeMorphed`: Protects specific containers ALWAYS, regardless of focus

### 4b. Conditional Polling

Use `hx-trigger` with JavaScript conditions to pause polling during user interaction:

```html
<div hx-get="/refresh-endpoint"
     hx-trigger="every 2s [document.activeElement.id!='my-input' && document.activeElement.id!='my-select']"
     hx-swap="morph:innerHTML">
```

The bracket expression is evaluated on each polling interval. If it returns `false`, the request is skipped entirely.

**When to use conditional polling vs `ignoreActive`:**
- `ignoreActive`: Morphs everything except the focused element (good for most cases)
- Conditional polling: Stops ALL refreshing while user interacts (better for complex multi-element forms where partial morph causes layout shifts)

**ANTI-PATTERN: Unconditional polling that refreshes content while user is actively typing or selecting** - even with `ignoreActive`, surrounding content changes can cause jarring UX. Use conditional polling for regions with intensive user interaction.

---

## Complete Setup Template

Copy-paste starting point for any auto-refreshing HTMX page:

```html
<!DOCTYPE html>
<html>
<head>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <script src="https://cdn.jsdelivr.net/npm/idiomorph@0.3.0/dist/idiomorph-ext.min.js"></script>
</head>
<body hx-ext="morph">
  <main>
    <div hx-get="/refresh"
         hx-trigger="every 2s [document.activeElement.id!='my-input']"
         hx-swap="morph:innerHTML">
      <!-- auto-refreshed content -->
    </div>
  </main>
  <script>
  /* Layer 1: Mandatory Idiomorph defaults */
  Idiomorph.defaults.ignoreActive = true;
  Idiomorph.defaults.ignoreActiveValue = true;

  /* Layer 3b: Preserve details/accordion open state */
  Idiomorph.defaults.callbacks.beforeAttributeUpdated = function(attr, node) {
    if (attr === 'open' && node.tagName === 'DETAILS') return false;
  };

  /* Layer 4a: Exclude specific containers from morphing (if needed) */
  Idiomorph.defaults.callbacks.beforeNodeMorphed = function(oldNode, newNode) {
    // if (oldNode.id === 'my-protected-container') return false;
    return true;
  };

  /* Layer 3c: Preserve scroll positions (update selectors as needed) */
  (function() {
    var scrollState = {}, selectors = '.scrollable-panel';
    document.body.addEventListener('htmx:beforeSwap', function() {
      scrollState = {};
      document.querySelectorAll(selectors).forEach(function(el, i) {
        scrollState[i] = el.scrollTop;
      });
    });
    document.body.addEventListener('htmx:afterSwap', function() {
      document.querySelectorAll(selectors).forEach(function(el, i) {
        if (scrollState[i] !== undefined) el.scrollTop = scrollState[i];
      });
    });
  })();
  </script>
</body>
</html>
```

---

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "I don't need morph swaps, my page is simple" | ANY auto-refresh destroys state. Even a simple status display with a nearby form will break. Use morph swaps. |
| "Plain innerHTML swap is simpler" | Simpler to write, broken for users. Morph swap is one attribute change. |
| "I'll add idiomorph later if users complain" | Users can't type, select, or scroll. They'll complain immediately. Add it from the start. |
| "ignoreActive handles everything" | It handles the focused element only. Details state, scroll position, and multi-element forms need separate handling. |
| "I need both idiomorph.min.js and idiomorph-ext.min.js" | No. The ext file includes everything. One import only. |
| "Conditional polling is overkill" | If users interact with content inside the polling region, conditional polling prevents jarring partial updates. |
| "I'll just use a longer polling interval" | Slower refresh doesn't fix state destruction - it just makes it less frequent. Use morph swaps. |
| "Idiomorph preserves scroll position automatically" | FALSE. Idiomorph morphs the DOM but does NOT preserve scrollTop. You MUST add htmx:beforeSwap/afterSwap handlers manually. This is the #1 misconception. |
| "I'll handle details state server-side" | Server-side round-tripping is fragile and complex. The `beforeAttributeUpdated` callback is one line and works universally. Use the callback. |
| "No custom JavaScript needed - morph handles everything" | Morph handles DOM diffing. It does NOT handle: ignoreActive defaults, details preservation, scroll preservation, or selective exclusion. These all require JavaScript configuration. |
| "I forgot hx-ext='morph' on body but morph:innerHTML still works" | It does NOT work. Without `hx-ext="morph"`, HTMX does not recognize the `morph:` prefix and falls back to plain innerHTML swap silently. |

## Red Flags - STOP

If you see any of these in your code, stop and fix immediately:

- `hx-swap="innerHTML"` on an auto-refreshing element (MUST be `morph:innerHTML`)
- `hx-swap="outerHTML"` on an auto-refreshing element (MUST be `morph:outerHTML` or `morph`)
- Both `idiomorph.min.js` AND `idiomorph-ext.min.js` in script tags
- Only `idiomorph.min.js` without the ext variant
- No `hx-ext="morph"` on `<body>` when using morph swaps
- Missing `Idiomorph.defaults.ignoreActive = true` on pages with forms/inputs
- `<details>` elements in auto-refreshed content without `beforeAttributeUpdated` callback
- Scrollable panels in auto-refreshed content without scroll preservation handlers
- Assuming "Idiomorph handles scroll position automatically" (it does NOT)
- Missing `hx-ext="morph"` on `<body>` while using `morph:innerHTML` swap (silently falls back to plain swap)

## Verification Checklist

Before shipping any auto-refreshing HTMX page:

- [ ] Single CDN import: `idiomorph-ext.min.js` only
- [ ] `hx-ext="morph"` on `<body>`
- [ ] `Idiomorph.defaults.ignoreActive = true` set
- [ ] `Idiomorph.defaults.ignoreActiveValue = true` set
- [ ] All auto-refresh swaps use `morph:innerHTML` or `morph:outerHTML`
- [ ] No plain `innerHTML`/`outerHTML` swaps on polling regions
- [ ] `<details>` elements protected with `beforeAttributeUpdated` callback
- [ ] Scrollable panels have scroll preservation handlers
- [ ] Forms tested: can type in inputs during refresh without losing text
- [ ] Forms tested: can open select boxes during refresh without them closing
- [ ] Conditional polling considered for intensive interaction areas

## Reference

- HTMX Idiomorph extension: https://htmx.org/extensions/idiomorph/
- Idiomorph GitHub (options/callbacks): https://github.com/bigskysoftware/idiomorph
- Real-world example: `cmd/sgai/templates/trees.html` (lines 1644-1673)

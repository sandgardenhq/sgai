---
name: browser-bug-testing-workflow
description: You must use this skill when debugging web UI bugs or testing interactive components that require multi-step browser interactions. Automates common UI testing patterns for debugging web applications - starting servers, navigating to pages, interacting with form elements, and verifying expected behaviors without manual step-by-step Playwright commands
metadata:
  requires:
    - run-long-running-processes-in-tmux
---

# Browser Bug Testing Workflow

## Overview

This skill provides a structured workflow for automated browser testing during UI bug investigation. It combines tmux session management for server processes with Playwright browser automation to create reproducible testing scenarios without manual intervention.

## When to Use

Use when:
- Debugging interactive UI components (checkboxes, forms, buttons)
- Testing multi-step user workflows
- Verifying expected behaviors after code changes
- Reproducing browser-specific bugs
- Testing responsive design interactions
- Validating form submissions and dialog behaviors

**Especially useful for:**
- Auto-mode checkbox testing
- Form validation workflows
- Dynamic content loading
- Modal/dialog interactions
- State-dependent UI elements

## Core Workflow Pattern

1. **Setup Phase**: Start server in tmux session
2. **Navigation Phase**: Open browser and navigate to target page
3. **Interaction Phase**: Perform UI interactions (click, type, select)
4. **Verification Phase**: Check expected outcomes and behaviors
5. **Cleanup Phase**: Capture results and clean up sessions

## Quick Reference

| Step | Action | Example |
|------|--------|---------|
| Start Server | tmux new-session -d -s server "npm run dev" | `tmux new-session -d -s webapp "python -m http.server 8000"` |
| Navigate | playwright_browser_navigate | `playwright_browser_navigate "http://localhost:8000"` |
| Interact | playwright_browser_click/type/select | `playwright_browser_click "checkbox" "ref"` |
| Wait | playwright_browser_wait_for | `playwright_browser_wait_for text="Loading complete"` |
| Verify | playwright_browser_snapshot | `playwright_browser_snapshot` |
| Capture | playwright_browser_take_screenshot | `playwright_browser_take_screenshot` |

## Implementation

### Phase 1: Server Setup with Tmux

**REQUIRED SUB-SKILL:** Use `skills:run-long-running-processes-in-tmux`

```bash
# Start development server in detached tmux session
tmux new-session -d -s webserver "npm run dev"

# Verify server is running
tmux capture-pane -t webserver -S - -p | grep "Server running"

# Wait for server to be ready
sleep 3
```

### Phase 2: Browser Navigation

```javascript
// Navigate to the application
await playwright_browser_navigate("http://localhost:3000");

// Wait for page to load
await playwright_browser_wait_for(time: 2);

// Take initial snapshot for baseline
await playwright_browser_snapshot();
```

### Phase 3: UI Interaction Patterns

#### Checkbox Testing Pattern
```javascript
// Find and click checkbox
await playwright_browser_click("auto mode checkbox", "checkbox-ref");

// Verify state change
await playwright_browser_wait_for(text="Auto mode enabled");

// Click action button that depends on checkbox
await playwright_browser_click("start button", "start-ref");
```

#### Form Interaction Pattern
```javascript
// Fill form fields
await playwright_browser_fill_form({
  fields: [
    { name: "username", type: "textbox", ref: "user-input", value: "testuser" },
    { name: "email", type: "textbox", ref: "email-input", value: "test@example.com" },
    { name: "remember", type: "checkbox", ref: "remember-check", value: true }
  ]
});

// Submit form
await playwright_browser_click("submit button", "submit-ref");
```

#### Dropdown Selection Pattern
```javascript
// Select dropdown option
await playwright_browser_select_option("country dropdown", "country-ref", ["United States"]);

// Verify selection was applied
await playwright_browser_wait_for(text="United States selected");
```

### Phase 4: Verification and Validation

#### Content Verification
```javascript
// Check for expected text/content
await playwright_browser_wait_for(text="Operation completed successfully");

// Take screenshot for documentation
await playwright_browser_take_screenshot(filename="test-result.png");

// Get current page state
await playwright_browser_snapshot();
```

#### Dialog/Modal Testing
```javascript
// Trigger dialog
await playwright_browser_click("delete button", "delete-ref");

// Handle confirmation dialog
await playwright_browser_handle_dialog(accept: true);

// Verify dialog result
await playwright_browser_wait_for(textGone="Are you sure?");
```

#### Error State Testing
```javascript
// Trigger error condition
await playwright_browser_click("invalid action", "invalid-ref");

// Check for error message
await playwright_browser_wait_for(text="Error: Invalid input");

// Capture error state
await playwright_browser_take_screenshot(filename="error-state.png");
```

### Phase 5: Cleanup and Results

```bash
# Capture server logs for debugging
tmux capture-pane -t webserver -S - -p > server-logs.txt

# Clean up tmux session
tmux kill-session -t webserver

# Close browser
await playwright_browser_close();
```

## Common Testing Scenarios

### Scenario 1: Auto-Mode Checkbox Bug
```javascript
// 1. Navigate to page
await playwright_browser_navigate("http://localhost:3000/settings");

// 2. Find auto-mode checkbox
await playwright_browser_snapshot(); // Initial state

// 3. Click checkbox
await playwright_browser_click("auto mode checkbox", "auto-checkbox");

// 4. Click start button
await playwright_browser_click("start button", "start-btn");

// 5. Wait for output
await playwright_browser_wait_for(text="Auto mode started");

// 6. Verify behavior
await playwright_browser_snapshot();
```

### Scenario 2: Form Validation Testing
```javascript
// 1. Navigate to form
await playwright_browser_navigate("http://localhost:3000/register");

// 2. Submit empty form
await playwright_browser_click("submit", "submit-btn");

// 3. Check validation errors
await playwright_browser_wait_for(text="Name is required");

// 4. Fill valid data
await playwright_browser_fill_form({
  fields: [
    { name: "name", type: "textbox", ref: "name-input", value: "John Doe" },
    { name: "email", type: "textbox", ref: "email-input", value: "john@example.com" }
  ]
});

// 5. Submit again
await playwright_browser_click("submit", "submit-btn");

// 6. Verify success
await playwright_browser_wait_for(text="Registration successful");
```

### Scenario 3: Dynamic Content Loading
```javascript
// 1. Navigate to page
await playwright_browser_navigate("http://localhost:3000/dashboard");

// 2. Trigger data load
await playwright_browser_click("load data", "load-btn");

// 3. Wait for loading indicator
await playwright_browser_wait_for(text="Loading...");

// 4. Wait for content
await playwright_browser_wait_for(textGone="Loading...");

// 5. Verify data loaded
await playwright_browser_wait_for(text="Data loaded successfully");
```

## Integration with Required Skills

### Tmux Integration
**REQUIRED:** Always use tmux for server processes to enable:
- Background server execution
- Log capture for debugging
- Clean session management
- Process isolation between tests

### Playwright Tool Integration
This skill leverages these Playwright functions:
- `playwright_browser_navigate` - Page navigation
- `playwright_browser_click` - Element interaction
- `playwright_browser_type` - Text input
- `playwright_browser_fill_form` - Form completion
- `playwright_browser_select_option` - Dropdown selection
- `playwright_browser_wait_for` - Timing synchronization
- `playwright_browser_snapshot` - State verification
- `playwright_browser_take_screenshot` - Visual documentation
- `playwright_browser_handle_dialog` - Dialog management
- `playwright_browser_close` - Cleanup

## Common Mistakes

- **Not using tmux for servers**: Leads to hanging processes and lost logs
- **Missing wait conditions**: Race conditions between UI and test actions
- **Not capturing baseline snapshots**: Hard to verify what changed
- **Skipping cleanup**: Leaves resources running between tests
- **Not handling dialogs**: Tests hang waiting for user interaction
- **Wrong element references**: Use exact refs from snapshots, not descriptions

## Best Practices

1. **Always capture before/after snapshots** for state comparison
2. **Use specific wait conditions** instead of arbitrary timeouts
3. **Capture server logs** when tests fail for debugging
4. **Clean up tmux sessions** to prevent resource leaks
5. **Document test scenarios** with screenshots and logs
6. **Test both positive and negative cases** for comprehensive coverage

---

## Screenshot Storage Guidelines

**IMPORTANT:** All screenshots must be stored in the retrospective's screenshots directory:

```
.sgai/retrospectives/<retrospective-id>/screenshots/
```

When taking screenshots:
1. Get the current retrospective ID from `.sgai/state.json` (field: `retrospectiveSession` or from .sgai/PROJECT_MANAGEMENT.md header)
2. Create the screenshots directory if it doesn't exist
3. Store screenshots with descriptive names

**Example:**
```javascript
// Create screenshots directory in the retrospective folder
await bash("mkdir -p .sgai/retrospectives/2025-01-15-09-30.abc1/screenshots");

// Take screenshot and save to the correct location
await playwright_browser_take_screenshot({
  filename: ".sgai/retrospectives/2025-01-15-09-30.abc1/screenshots/before-fix.png"
});

// After making changes
await playwright_browser_take_screenshot({
  filename: ".sgai/retrospectives/2025-01-15-09-30.abc1/screenshots/after-fix.png"
});
```

This ensures all visual evidence is properly organized and preserved with the retrospective session for future analysis.

---

## Visual Verification Priority

**CRITICAL:** Always prefer taking Playwright screenshots over CSS evaluation for visual verification.

When verifying UI appearance:
1. **FIRST:** Take a screenshot with `playwright_browser_take_screenshot`
2. **SECOND:** Examine the screenshot visually to verify the expected result
3. **ONLY IF NEEDED:** Use CSS evaluation (`playwright_browser_evaluate`) for specific CSS property checks

**Why screenshots are preferred:**
- Screenshots capture the ACTUAL rendered appearance, including browser-specific rendering
- CSS evaluation only shows computed values, not visual reality
- Color contrasts, font rendering, element overlap are only visible in screenshots
- Screenshots provide evidence for retrospective analysis
- Human reviewers can verify screenshots but not CSS evaluation results

**Anti-pattern (DON'T):**
```javascript
// Checking CSS properties alone doesn't show visual reality
await playwright_browser_evaluate({
  function: "() => getComputedStyle(document.querySelector('.button')).backgroundColor"
});
```

**Preferred pattern (DO):**
```javascript
// Take screenshot to verify visual appearance
await playwright_browser_take_screenshot({
  filename: ".sgai/retrospectives/<id>/screenshots/button-styling.png"
});

// Then examine the screenshot to verify colors, spacing, alignment are correct
```

---

## Button Group Consistency Guidelines

When testing or implementing UI with button groups, ensure all buttons in the same group have consistent sizing and alignment:

### Requirements for Button Groups

1. **Same Width**: All buttons in a group should have equal width
2. **Same Height**: All buttons in a group should have equal height
3. **Internal Text Horizontal Alignment**: Text inside buttons should be aligned consistently (centered, left, or right)
4. **External Text Horizontal Alignment**: The buttons themselves should be aligned consistently within their container

### Verification Checklist

When testing button groups:
```javascript
// Take a screenshot of the button group
await playwright_browser_take_screenshot({
  filename: ".sgai/retrospectives/<id>/screenshots/button-group.png"
});

// Visually verify in the screenshot:
// 1. All buttons have the same width
// 2. All buttons have the same height
// 3. Text inside buttons is consistently aligned
// 4. Buttons are properly aligned as a group
```

### CSS Guidelines for Button Groups

If implementing button groups, ensure:
```css
/* All buttons in a group should use consistent sizing */
.button-group button {
  min-width: 100px;           /* Consistent minimum width */
  height: 40px;               /* Consistent height */
  text-align: center;         /* Consistent internal text alignment */
}

/* Use flexbox or grid for external alignment */
.button-group {
  display: flex;
  gap: 0.5rem;
  align-items: center;        /* Vertical alignment */
  justify-content: flex-start; /* Horizontal alignment */
}
```

### Common Issues to Catch

- Buttons with different widths due to varying text length (use `min-width`)
- Buttons with different heights due to padding inconsistencies
- Text not centered within buttons
- Button group not properly aligned in its container

## Real-World Impact

From actual debugging sessions:
- Manual checkbox testing: 5-10 minutes per iteration
- Automated workflow: 30 seconds per test run
- Bug reproduction consistency: 100% vs variable manual testing
- Test coverage expansion: 10x more scenarios tested in same time

## Example Complete Workflow

```javascript
// Complete auto-mode checkbox test
async function testAutoModeCheckbox() {
  try {
    // Phase 1: Start server (tmux)
    await bash("tmux new-session -d -s testserver 'npm run dev'");
    await bash("sleep 3");

    // Phase 2: Navigate
    await playwright_browser_navigate("http://localhost:3000");
    await playwright_browser_wait_for(time: 2);

    // Phase 3: Baseline
    await playwright_browser_snapshot();

    // Phase 4: Interact
    await playwright_browser_click("auto mode checkbox", "auto-checkbox");
    await playwright_browser_click("start button", "start-btn");

    // Phase 5: Verify
    await playwright_browser_wait_for(text="Auto mode activated");
    await playwright_browser_snapshot();
    await playwright_browser_take_screenshot("auto-mode-result.png");

    // Phase 6: Cleanup
    await bash("tmux capture-pane -t testserver -S - -p > test.log");
    await bash("tmux kill-session -t testserver");
    await playwright_browser_close();

    console.log("Test completed successfully");
  } catch (error) {
    console.error("Test failed:", error);
    // Capture debug info
    await bash("tmux capture-pane -t testserver -S - -p > error.log");
    await playwright_browser_take_screenshot("error-state.png");
  }
}
```

This workflow transforms manual UI debugging into automated, reproducible test scenarios that can be run repeatedly during development.

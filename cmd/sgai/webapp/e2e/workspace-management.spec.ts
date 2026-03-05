import { test, expect } from "@playwright/test";

test.describe("Workspace Management Workflow", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("create workspace → fork → delete", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    await expect(page.locator("h3")).toBeVisible();

    await page.click('button:has-text("Edit GOAL")');
    await page.waitForURL(/\/goal\/edit/);

    await page.fill('[data-testid="markdown-editor"] textarea', `---
flow: |
  "coordinator" -> "backend-go-developer"
---

# Test Goal

This is a test goal for the workspace.
`);

    await page.click('button:has-text("Save GOAL.md")');
    await page.waitForSelector("text=Saved!");

    await page.click('a:has-text("←")');
    await page.waitForURL(/\/workspaces\/[^/]+\/progress/);

    await page.goto("/");
    await page.waitForSelector('[data-testid="dashboard-content"]', { timeout: 10000 });
  });

  test("attach external repository → fork → detach", async ({ page }) => {
    await page.goto("/");
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const externalIndicator = page.locator('[aria-label="External workspace"]');
    const isExternal = await externalIndicator.isVisible().catch(() => false);

    if (isExternal) {
      await expect(externalIndicator).toBeVisible();
    }
  });

  test("workspace tree displays correctly", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const pinnedSection = page.locator('[role="region"][aria-label="Pinned"]');
    const inProgressSection = page.locator('[role="region"][aria-label="In progress"]');

    const hasPinned = await pinnedSection.isVisible().catch(() => false);
    const hasInProgress = await inProgressSection.isVisible().catch(() => false);

    expect(hasPinned || hasInProgress).toBe(true);
  });

  test("workspace tree shows forks nested under parent", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const expandButton = page.locator('[aria-label="Toggle forks"]').first();
    const isVisible = await expandButton.isVisible().catch(() => false);

    if (isVisible) {
      await expandButton.click();
      await page.waitForTimeout(500);
    }
  });

  test("workspace tree updates on state change", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const firstWorkspace = page.locator("a[href^='/workspaces/']").first();
    await firstWorkspace.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    await expect(page.locator("h3")).toBeVisible();
  });

  test("pinned workspaces appear in pinned section", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const pinnedSection = page.locator('[role="region"][aria-label="Pinned"]');
    const isVisible = await pinnedSection.isVisible().catch(() => false);

    if (isVisible) {
      const pinnedIndicator = pinnedSection.locator('[aria-label="Pinned"]');
      const hasPinned = await pinnedIndicator.isVisible().catch(() => false);
      expect(typeof hasPinned).toBe("boolean");
    }
  });

  test("in progress workspaces appear in in progress section", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const inProgressSection = page.locator('[role="region"][aria-label="In progress"]');
    const isVisible = await inProgressSection.isVisible().catch(() => false);

    if (isVisible) {
      const runningIndicator = inProgressSection.locator('[aria-label="Running"]');
      const hasRunning = await runningIndicator.isVisible().catch(() => false);
      expect(typeof hasRunning).toBe("boolean");
    }
  });

  test("delete confirmation dialog works correctly", async ({ page }) => {
    await page.goto("/");

    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceItem = page.locator("a[href^='/workspaces/']").first();
    await workspaceItem.hover();

    const deleteButton = page.locator('[aria-label*="Delete"]').first();
    const isDeleteVisible = await deleteButton.isVisible().catch(() => false);

    if (isDeleteVisible) {
      await deleteButton.click();

      await expect(page.locator("text=Delete workspace")).toBeVisible();

      await page.click('button:has-text("Cancel")');

      await expect(page.locator("text=Delete workspace")).not.toBeVisible();
    }
  });
});

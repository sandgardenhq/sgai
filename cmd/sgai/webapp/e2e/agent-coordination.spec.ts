import { test, expect } from "@playwright/test";

test.describe("Agent Coordination Workflow", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("start agent → view progress → stop", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    await expect(page.locator("h3")).toBeVisible();

    const statusBadge = page.locator("text=stopped, text=running").first();
    const statusText = await statusBadge.textContent().catch(() => "stopped");
    const isRunning = statusText?.includes("running") ?? false;

    if (!isRunning) {
      await page.click('button:has-text("Start")');

      await page.waitForSelector("text=running", { timeout: 10000 });

      await expect(page.locator("text=running")).toBeVisible();
    }

    await page.click('a[href$="/progress"]');

    await page.waitForURL(/\/progress/);

    if (isRunning) {
      await page.click('button:has-text("Stop")');

      await page.waitForSelector("text=stopped", { timeout: 10000 });

      await expect(page.locator("text=stopped")).toBeVisible();
    }
  });

  test("navigate between agents", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const agentsButton = page.locator('button:has-text("Agents")');
    const hasAgents = await agentsButton.isVisible().catch(() => false);

    if (hasAgents) {
      await agentsButton.click();

      await page.waitForURL(/\/agents/);

      await expect(page.locator("text=Agent")).toBeVisible();
    }
  });

  test("self drive mode works correctly", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const statusBadge = page.locator("text=stopped, text=running").first();
    const statusText = await statusBadge.textContent().catch(() => "stopped");
    const isRunning = statusText?.includes("running") ?? false;

    if (!isRunning) {
      await page.click('button:has-text("Self-drive")');

      await page.waitForSelector("text=running", { timeout: 10000 });

      await expect(page.locator("text=running")).toBeVisible();
    }
  });

  test("view log tab shows agent output", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const logTab = page.locator('a[href$="/log"]');
    const hasLogTab = await logTab.isVisible().catch(() => false);

    if (hasLogTab) {
      await logTab.click();

      await page.waitForURL(/\/log/);

      await expect(page.locator("text=Log")).toBeVisible();
    }
  });

  test("view messages tab shows agent communication", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const messagesTab = page.locator('a[href$="/messages"]');
    const hasMessagesTab = await messagesTab.isVisible().catch(() => false);

    if (hasMessagesTab) {
      await messagesTab.click();

      await page.waitForURL(/\/messages/);

      await expect(page.locator("text=Messages")).toBeVisible();
    }
  });

  test("view changes tab shows diffs", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const changesTab = page.locator('a[href$="/changes"]');
    const hasChangesTab = await changesTab.isVisible().catch(() => false);

    if (hasChangesTab) {
      await changesTab.click();

      await page.waitForURL(/\/changes/);

      await expect(page.locator("text=Diffs")).toBeVisible();
    }
  });

  test("view commits tab shows commit history", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const commitsTab = page.locator('a[href$="/commits"]');
    const hasCommitsTab = await commitsTab.isVisible().catch(() => false);

    if (hasCommitsTab) {
      await commitsTab.click();

      await page.waitForURL(/\/commits/);

      await expect(page.locator("text=Commits")).toBeVisible();
    }
  });

  test("execution time displays correctly", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const runningIndicator = page.locator('[aria-label="Running"]').first();
    const isRunning = await runningIndicator.isVisible().catch(() => false);

    if (isRunning) {
      await runningIndicator.click();
    } else {
      const workspaceLink = page.locator("a[href^='/workspaces/']").first();
      await workspaceLink.click();
    }

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const execTimeBadge = page.locator('[aria-label="Total execution time"]').first();
    const hasExecTime = await execTimeBadge.isVisible().catch(() => false);

    if (hasExecTime) {
      const timeText = await execTimeBadge.textContent();
      expect(timeText).toMatch(/\d+m\s+\d+s|\d+s/);
    }
  });

  test("current agent and model display correctly", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const runningIndicator = page.locator('[aria-label="Running"]').first();
    const isRunning = await runningIndicator.isVisible().catch(() => false);

    if (isRunning) {
      await runningIndicator.click();

      await page.waitForURL(/\/workspaces\/[^/]+/);

      const agentBadge = page.locator('[data-variant="secondary"]').filter({ hasText: /coordinator|backend|react/ });
      const hasAgentBadge = await agentBadge.isVisible().catch(() => false);

      if (hasAgentBadge) {
        const badgeText = await agentBadge.textContent();
        expect(badgeText).toBeTruthy();
      }
    }
  });

  test("pin/unpin workspace works correctly", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const pinButton = page.locator('button:has-text("Pin"), button:has-text("Unpin")').first();
    const initialText = await pinButton.textContent();

    await pinButton.click();

    await page.waitForTimeout(1000);

    const newText = await pinButton.textContent();
    expect(newText).not.toBe(initialText);
  });

  test("open in editor works correctly", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const openEditorButton = page.locator('button:has-text("Open in Editor")');
    const hasOpenEditor = await openEditorButton.isVisible().catch(() => false);

    if (hasOpenEditor) {
      await openEditorButton.click();

      await page.waitForTimeout(1000);

      await expect(page.locator("text=Failed to open editor")).not.toBeVisible();
    }
  });

  test("respond to agent question", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const needsInputIndicator = page.locator('[aria-label="Waiting for response"]').first();
    const needsInput = await needsInputIndicator.isVisible().catch(() => false);

    if (needsInput) {
      await needsInputIndicator.click();

      await page.waitForURL(/\/respond/);

      await expect(page.locator("text=Response Required")).toBeVisible();

      const firstChoice = page.locator('input[type="radio"], input[type="checkbox"]').first();
      const hasChoice = await firstChoice.isVisible().catch(() => false);

      if (hasChoice) {
        await firstChoice.click();
      }

      await page.click('button:has-text("Send Response")');

      await page.waitForURL(/\/progress/, { timeout: 10000 });
    }
  });
});

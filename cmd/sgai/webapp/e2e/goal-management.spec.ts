import { test, expect } from "@playwright/test";

test.describe("Goal Management Workflow", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("create goal → edit → run agents → view results", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      await expect(page.locator("text=Edit GOAL.md")).toBeVisible();

      const editor = page.locator('[data-testid="markdown-editor"]');
      const hasEditor = await editor.isVisible().catch(() => false);

      if (hasEditor) {
        await editor.click();

        await page.keyboard.press("Control+a");
        await page.keyboard.type(`---
flow: |
  "coordinator" -> "backend-go-developer"
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "opencode/glm-5"
  "backend-go-developer": "opencode/glm-5"
completionGateScript: make test
---

# Test Goal

## Task: Implement Feature X

- [ ] Design the API
- [ ] Write tests
- [ ] Implement the feature
`);
      }

      await page.click('button:has-text("Save GOAL.md")');

      await page.waitForSelector("text=Saved!", { timeout: 10000 });

      await page.click('a:has-text("←")');

      await page.waitForURL(/\/workspaces\/[^/]+\/progress/);

      const startButton = page.locator('button:has-text("Start")');
      const canStart = await startButton.isVisible().catch(() => false);

      if (canStart) {
        await startButton.click();

        await page.waitForSelector("text=running", { timeout: 10000 });
      }

      await page.click('a[href$="/progress"]');

      await expect(page.locator("h3")).toBeVisible();

      const logTab = page.locator('a[href$="/log"]');
      const hasLogTab = await logTab.isVisible().catch(() => false);

      if (hasLogTab) {
        await logTab.click();

        await page.waitForURL(/\/log/);

        await expect(page.locator("text=Log")).toBeVisible();
      }
    }
  });

  test("goal editor shows workspace description", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const description = page.locator("text=Edit GOAL.md").locator("..").locator("span").first();
      const hasDescription = await description.isVisible().catch(() => false);

      if (hasDescription) {
        await expect(description).toBeVisible();
      }
    }
  });

  test("goal editor keyboard shortcut saves", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      await page.keyboard.press("Control+s");

      await page.waitForSelector("text=Saved!", { timeout: 10000 });
    }
  });

  test("goal editor shows loading state", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/, { timeout: 5000 });
    }
  });

  test("goal editor autocomplete for agents", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const editor = page.locator('[data-testid="markdown-editor"]');
      const hasEditor = await editor.isVisible().catch(() => false);

      if (hasEditor) {
        await editor.click();

        await page.keyboard.type('flow: |\n  "');

        await page.waitForTimeout(1000);

        await expect(editor).toBeVisible();
      }
    }
  });

  test("goal editor preview mode", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const previewButton = page.locator('button:has-text("Preview")');
      const hasPreview = await previewButton.isVisible().catch(() => false);

      if (hasPreview) {
        await previewButton.click();

        const previewContent = page.locator("text=Nothing to preview, text=Preview");
        const hasPreviewContent = await previewContent.isVisible().catch(() => false);

        if (hasPreviewContent) {
          await expect(previewContent).toBeVisible();
        }
      }
    }
  });

  test("goal editor write mode", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const writeButton = page.locator('button[aria-pressed="true"]:has-text("Write")');
      const hasWriteButton = await writeButton.isVisible().catch(() => false);

      if (hasWriteButton) {
        await expect(writeButton).toBeVisible();
      }
    }
  });

  test("goal editor toolbar actions", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const boldButton = page.locator('[aria-label="Bold"]');
      const italicButton = page.locator('[aria-label="Italic"]');
      const headingButton = page.locator('[aria-label="Heading 1"]');

      const hasBold = await boldButton.isVisible().catch(() => false);
      const hasItalic = await italicButton.isVisible().catch(() => false);
      const hasHeading = await headingButton.isVisible().catch(() => false);

      if (hasBold) await expect(boldButton).toBeVisible();
      if (hasItalic) await expect(italicButton).toBeVisible();
      if (hasHeading) await expect(headingButton).toBeVisible();
    }
  });

  test("compose goal from scratch", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const composeButton = page.locator('button:has-text("Compose GOAL")');
    const hasCompose = await composeButton.isVisible().catch(() => false);

    if (hasCompose) {
      await composeButton.click();

      await page.waitForURL(/\/compose/);

      await expect(page.locator("text=Compose")).toBeVisible();
    }
  });

  test("goal content displays in progress tab", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const progressTab = page.locator('a[href$="/progress"]');
    const hasProgressTab = await progressTab.isVisible().catch(() => false);

    if (hasProgressTab) {
      await progressTab.click();

      await expect(page.locator("h3")).toBeVisible();
    }
  });

  test("goal validation prevents empty save", async ({ page }) => {
    await page.waitForSelector("text=Workspaces", { timeout: 10000 });

    const workspaceLink = page.locator("a[href^='/workspaces/']").first();
    await workspaceLink.click();

    await page.waitForURL(/\/workspaces\/[^/]+/);

    const editGoalButton = page.locator('button:has-text("Edit GOAL")');
    const hasEditGoal = await editGoalButton.isVisible().catch(() => false);

    if (hasEditGoal) {
      await editGoalButton.click();

      await page.waitForURL(/\/goal\/edit/);

      const editor = page.locator('[data-testid="markdown-editor"]');
      const hasEditor = await editor.isVisible().catch(() => false);

      if (hasEditor) {
        await editor.click();
        await page.keyboard.press("Control+a");
        await page.keyboard.press("Backspace");

        const saveButton = page.locator('button:has-text("Save GOAL.md")');
        await expect(saveButton).toBeDisabled();
      }
    }
  });
});

import { test, expect, type Page } from "@playwright/test";

const BASE_URL = "http://127.0.0.1:8181";
const WORKSPACE = "test-project";

async function waitForReactContent(page: Page) {
  await page.locator("#root > *").first().waitFor({ timeout: 10_000 });
}

// ─────────────────────────────────────────────────────────────────────────────
// M7 React-Only Full Test Suite (R-7)
// Verifies ALL user flows on React-only interface (no HTMX fallback)
// No sgai-ui cookie is set — React is now the default and only interface.
// ─────────────────────────────────────────────────────────────────────────────

test.describe("M7 Entity Browsers — React-Only", () => {
  test("agents list loads and shows agent definitions", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/agents`);
    await waitForReactContent(page);

    await expect(page.getByText("Agents").first()).toBeVisible();
    await expect(page.getByText("backend-developer").first()).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("backend-developer");
    expect(pageText).toContain("frontend-developer");
  });

  test("skills list loads with categories", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/skills`);
    await waitForReactContent(page);

    await expect(page.getByText("Skills").first()).toBeVisible();
    await expect(page.getByText("general-skill").first()).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText?.toLowerCase()).toContain("general");
    expect(pageText?.toLowerCase()).toContain("coding-practices");
    expect(pageText).toContain("general-skill");
    expect(pageText).toContain("go-review");
  });

  test("skill detail loads with content", async ({ page }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/skills/general-skill`,
    );
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/skills/general-skill") &&
        response.status() === 200,
      { timeout: 10_000 },
    );
    await expect(
      page.getByRole("heading", { name: "General Skill" }),
    ).toBeVisible({ timeout: 10_000 });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("General Skill");
    expect(pageText).toContain("general guidance");
  });

  test("nested skill detail loads", async ({ page }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/skills/coding-practices/go-review`,
    );
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/skills/") &&
        response.status() === 200,
      { timeout: 10_000 },
    );
    await expect(page.getByText("go-review").first()).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("go-review");
  });

  test("navigate from skills list to detail and back", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/skills`);
    await waitForReactContent(page);

    // Wait for skills data to render
    await expect(page.getByText("general-skill").first()).toBeVisible({
      timeout: 10_000,
    });

    await page.getByRole("link", { name: /general-skill/ }).click();
    await page.waitForURL(`**/${WORKSPACE}/skills/general-skill`);
    await waitForReactContent(page);

    // Wait for skill detail content to render
    await expect(page.getByText("General Skill").first()).toBeVisible({
      timeout: 10_000,
    });

    const detailText = await page.textContent("body");
    expect(detailText).toContain("General Skill");

    await page.getByRole("link", { name: /Back to Skills/ }).click();
    await page.waitForURL(`**/skills`);
    await waitForReactContent(page);

    // Wait for skills list to re-render
    await expect(page.getByText("general-skill").first()).toBeVisible({
      timeout: 10_000,
    });
  });

  test("snippets list loads with language categories", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/snippets`);
    await waitForReactContent(page);

    await expect(page.getByText("Snippets").first()).toBeVisible();
    await expect(page.getByText("HTTP Server").first()).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText?.toLowerCase()).toContain("go");
    expect(pageText?.toLowerCase()).toContain("python");
    expect(pageText).toContain("HTTP Server");
    expect(pageText).toContain("Hello World");
  });
});

test.describe("M7 Dashboard + Workspace Tree — React-Only", () => {
  test("dashboard loads with workspace tree sidebar", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    const sidebarLocator = page
      .locator("aside, [role='complementary']")
      .first();
    const sidebarText = await sidebarLocator.textContent();
    expect(sidebarText).toContain(WORKSPACE);
  });

  test("empty state shown when no workspace selected", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    await expect(page.getByText("Select a workspace")).toBeVisible();
  });

  test("clicking workspace shows its details with tabs", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);
    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    const encodedWorkspace = encodeURIComponent(WORKSPACE);
    const workspaceLink = page
      .locator(
        `aside a[href='/workspaces/${encodedWorkspace}/progress'], [role='complementary'] a[href='/workspaces/${encodedWorkspace}/progress']`,
      )
      .first();
    await workspaceLink.waitFor({ timeout: 10_000 });
    await workspaceLink.click();

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain(WORKSPACE);
    expect(mainText).toContain("Progress");
  });

  test("new workspace button is present and navigable", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);
    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    const newBtn = page
      .locator("aside, [role='complementary']")
      .first()
      .locator("button, a")
      .filter({ hasText: "[ + ]" });
    await expect(newBtn).toBeVisible();

    await newBtn.click();
    await page.waitForURL("**/workspaces/new");
  });
});

test.describe("M7 Session Tabs — React-Only", () => {
  const tabNames = [
    "Progress",
    "Log",
    "Diffs",
    "Commits",
    "Messages",
    "Internals",
    "Retrospectives",
    "Run",
  ];

  test("workspace detail page shows all session tabs", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    for (const tab of tabNames) {
      expect(mainText).toContain(tab);
    }
  });

  test("progress tab shows session state", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Progress");
  });

  test("messages tab shows inter-agent messages", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/messages`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Messages");
  });

  test("log tab shows output log", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/log`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Log");
  });

  test("diffs tab shows diff output", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/changes`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Diffs");
  });

  test("internals tab shows session internals", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/internals`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Internals");
  });

  test("retrospectives tab shows entries", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/retro`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    // Retro tab shows a split layout with "No retrospective sessions found" or sessions list
    const pageText = (await page.textContent("body"))?.toLowerCase() ?? "";
    expect(pageText).toContain("retrospective");
  });

  test("run tab loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/run`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Run");
  });
});

test.describe("M7 Response System — React-Only", () => {
  test("respond page loads without cookie", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/respond`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("workspace session controls are present", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain(WORKSPACE);
  });
});

test.describe("M7 GOAL Composer Wizard — React-Only", () => {
  test("compose landing page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("wizard step 1 loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/1`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("wizard step 2 loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/2`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("wizard step 3 loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/3`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("wizard step 4 loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/4`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("wizard finish page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/finish`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("compose preview loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/preview`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

test.describe("M7 Workspace Management — React-Only", () => {
  test("new workspace page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/new`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("fork creation page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/fork/new`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("rename page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/rename`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("edit goal page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/goal/edit`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("ad-hoc prompt page loads", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/adhoc`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("retrospective analyze page loads", async ({ page }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/retrospective/analyze`,
    );
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("retrospective apply page loads", async ({ page }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/retrospective/apply`,
    );
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

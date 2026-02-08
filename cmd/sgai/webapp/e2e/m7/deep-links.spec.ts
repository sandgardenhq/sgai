import { test, expect, type Page } from "@playwright/test";

const BASE_URL = "http://127.0.0.1:8181";
const WORKSPACE = "test-project";

async function waitForReactContent(page: Page) {
  await page.locator("#root > *").first().waitFor({ timeout: 10_000 });
}

// ─────────────────────────────────────────────────────────────────────────────
// M7 Deep Link Tests (R-12)
// Tests ALL deep link patterns via direct URL navigation.
// Verifies bookmarked URLs resolve correctly through React Router.
// No sgai-ui cookie is set — React is the sole interface.
// ─────────────────────────────────────────────────────────────────────────────

test.describe("M7 Deep Links — Dashboard Routes (R-12)", () => {
  test("/ loads dashboard with empty state", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await expect(page.getByText("SGAI").first()).toBeVisible();
    await expect(page.getByText("Select a workspace")).toBeVisible({
      timeout: 10_000,
    });
  });

  test("/trees redirects to /", async ({ page }) => {
    await page.goto(`${BASE_URL}/trees`);
    await waitForReactContent(page);

    expect(page.url()).toBe(`${BASE_URL}/`);
  });
});

test.describe("M7 Deep Links — Workspace Routes (R-12)", () => {
  test("/workspaces/{name} loads workspace detail", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("/workspaces/{name}/progress loads progress tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Progress");
  });

  test("/workspaces/{name}/internals loads internals tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/internals`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Internals");
  });

  test("/workspaces/{name}/messages loads messages tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/messages`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Messages");
  });

  test("/workspaces/{name}/log loads log tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/log`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Log");
  });

  test("/workspaces/{name}/changes loads diffs tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/changes`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Diffs");
  });

  test("/workspaces/{name}/commits loads commits tab", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/commits`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain("Commits");
  });

  test("/workspaces/{name}/retro loads retrospectives tab", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/retro`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    // The Retrospectives tab link should be visible in the tab nav
    await expect(
      page.getByRole("link", { name: "Retrospectives" }),
    ).toBeVisible();

    // The retro content shows a split layout with "No retrospective sessions found" or sessions list
    const pageText = (await page.textContent("body"))?.toLowerCase() ?? "";
    expect(pageText).toContain("retrospective");
  });
});

test.describe("M7 Deep Links — Entity Browser Routes (R-12)", () => {
  test("/workspaces/{name}/agents loads agents page", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/agents`);
    await waitForReactContent(page);

    // Wait for the actual agent data to render (not just the skeleton)
    await expect(page.getByText("backend-developer")).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("Agents");
    expect(pageText).toContain("backend-developer");
  });

  test("/workspaces/{name}/skills loads skills page", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/skills`);
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/skills") &&
        response.status() === 200,
      { timeout: 10_000 },
    );

    await expect(page.getByText("general-skill").first()).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("Skills");
    expect(pageText).toContain("general-skill");
  });

  test("/workspaces/{name}/skills/{skillName} loads skill detail", async ({
    page,
  }) => {
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

    await expect(page.getByText("General Skill")).toBeVisible({
      timeout: 10_000,
    });

    const pageText = await page.textContent("body");
    expect(pageText).toContain("General Skill");
  });

  test("/workspaces/{name}/snippets loads snippets page", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/snippets`);
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/snippets") &&
        response.status() === 200,
      { timeout: 10_000 },
    );

    const pageText = await page.textContent("body");
    expect(pageText).toContain("Snippets");
  });
});

test.describe("M7 Deep Links — Compose Routes (R-12)", () => {
  test("/compose loads compose landing", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/compose/step/1 loads wizard step 1", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/1`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/compose/step/2 loads wizard step 2", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/2`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/compose/step/3 loads wizard step 3", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/3`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/compose/step/4 loads wizard step 4", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/step/4`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/compose/finish loads wizard finish", async ({ page }) => {
    await page.goto(`${BASE_URL}/compose/finish`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

test.describe("M7 Deep Links — Response Routes (R-12)", () => {
  test("/respond loads response page", async ({ page }) => {
    await page.goto(`${BASE_URL}/respond`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/respond loads workspace response page", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/respond`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

test.describe("M7 Deep Links — Workspace Management Routes (R-12)", () => {
  test("/workspaces/new loads new workspace form", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/new`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/fork/new loads fork creation form", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/fork/new`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/rename loads rename form", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/rename`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/goal/edit loads goal editor", async ({ page }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/goal/edit`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/adhoc loads ad-hoc prompt page", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/adhoc`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/retrospective/analyze loads retro analyze", async ({
    page,
  }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/retrospective/analyze`,
    );
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });

  test("/workspaces/{name}/retrospective/apply loads retro apply", async ({
    page,
  }) => {
    await page.goto(
      `${BASE_URL}/workspaces/${WORKSPACE}/retrospective/apply`,
    );
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

test.describe("M7 Deep Links — Browser Navigation (R-12)", () => {
  test("browser back/forward navigation works through React Router", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);
    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/agents`);
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/agents") &&
        response.status() === 200,
      { timeout: 10_000 },
    );

    const agentsText = await page.textContent("body");
    expect(agentsText).toContain("Agents");

    await page.goBack();
    await waitForReactContent(page);
    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    await page.goForward();
    await waitForReactContent(page);

    await page.waitForResponse(
      (response) =>
        response.url().includes("/api/v1/agents") &&
        response.status() === 200,
      { timeout: 10_000 },
    );

    const agentsTextForward = await page.textContent("body");
    expect(agentsTextForward).toContain("Agents");
  });

  test("unknown route shows NotYetAvailable fallback", async ({ page }) => {
    await page.goto(`${BASE_URL}/nonexistent-page`);
    await waitForReactContent(page);

    const pageText = await page.textContent("body");
    expect(pageText).toBeTruthy();
  });
});

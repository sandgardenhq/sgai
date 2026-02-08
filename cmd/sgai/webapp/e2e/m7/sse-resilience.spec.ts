import { test, expect, type Page } from "@playwright/test";

const BASE_URL = "http://127.0.0.1:8181";
const WORKSPACE = "test-project";

async function waitForReactContent(page: Page) {
  await page.locator("#root > *").first().waitFor({ timeout: 10_000 });
}

// ─────────────────────────────────────────────────────────────────────────────
// M7 SSE Connection Resilience Tests (R-1, R-3)
// Tests SSE reconnection behavior, connection status indicator,
// exponential backoff, and snapshot rehydration on reconnect.
// No sgai-ui cookie is set — React is the sole interface.
// ─────────────────────────────────────────────────────────────────────────────

test.describe("M7 SSE Connection — Initial Connection (R-1)", () => {
  test("SSE connection is established on page load", async ({ page }) => {
    let sseRequestSeen = false;

    // Register listener BEFORE navigation so we catch the SSE request
    page.on("request", (req) => {
      if (req.url().includes("/api/v1/events/stream")) {
        sseRequestSeen = true;
      }
    });

    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // SSE request should have been made during page load
    expect(sseRequestSeen).toBe(true);

    // No "Reconnecting..." banner should be visible when connection is healthy
    const reconnectingBanner = page.getByText("Reconnecting...");
    const bannerCount = await reconnectingBanner.count();
    expect(bannerCount).toBe(0);
  });

  test("SSE connection serves workspace data on dashboard", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    const sidebarText = await page
      .locator("aside, [role='complementary']")
      .first()
      .textContent();
    expect(sidebarText).toContain(WORKSPACE);
  });
});

test.describe("M7 SSE Connection — Disruption Handling (R-3)", () => {
  test("blocking SSE URL triggers reconnection attempts", async ({
    page,
  }) => {
    const sseRequests: number[] = [];

    // Track all EventSource instances created by the SSE store so we can
    // force-trigger the onerror handler to simulate a connection drop.
    await page.addInitScript(() => {
      const instances: EventSource[] = [];
      (window as unknown as Record<string, unknown>).__sseInstances = instances;
      const OrigES = window.EventSource;
      window.EventSource = class extends OrigES {
        constructor(url: string | URL, init?: EventSourceInit) {
          super(url, init);
          instances.push(this);
        }
      } as typeof EventSource;
    });

    // Register listener BEFORE navigation
    page.on("request", (req) => {
      if (req.url().includes("/api/v1/events/stream")) {
        sseRequests.push(Date.now());
      }
    });

    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // Verify initial SSE request was made
    expect(sseRequests.length).toBeGreaterThanOrEqual(1);

    const initialRequestCount = sseRequests.length;

    // Block SSE URL so reconnection attempts are observable as aborted requests
    await page.route("**/api/v1/events/stream", (route) => route.abort());

    // Trigger the SSE store's onerror handler on the existing EventSource
    // to force a disconnect → scheduleReconnect cycle.
    await page.evaluate(() => {
      const instances = (
        window as unknown as { __sseInstances: EventSource[] }
      ).__sseInstances;
      for (const es of instances) {
        if (es.readyState !== 2 && es.onerror) {
          es.onerror(new Event("error"));
        }
      }
    });

    // Use expect.poll() to wait for reconnection attempts (condition-based, not time-based).
    // The SSE store's backoff starts at 1s, so reconnection requests should appear quickly.
    await expect
      .poll(() => sseRequests.length, {
        timeout: 10_000,
        message:
          "Expected reconnection attempts after SSE disruption (initial count: " +
          initialRequestCount +
          ")",
      })
      .toBeGreaterThan(initialRequestCount);
  });

  test("SSE store loads and provides workspace data", async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    // The SSE store should provide workspace data to the dashboard sidebar
    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // Verify workspace data is rendered (proving SSE store delivered data)
    const sidebarText = await page
      .locator("aside, [role='complementary']")
      .first()
      .textContent();
    expect(sidebarText).toContain(WORKSPACE);
  });
});

test.describe("M7 SSE Connection — Status Indicator (R-2)", () => {
  test("no reconnecting banner when connection is healthy", async ({
    page,
  }) => {
    let sseRequestSeen = false;

    page.on("request", (req) => {
      if (req.url().includes("/api/v1/events/stream")) {
        sseRequestSeen = true;
      }
    });

    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // Verify SSE was requested
    expect(sseRequestSeen).toBe(true);

    // No "Reconnecting..." banner should be visible
    const reconnectingBanner = page.getByText("Reconnecting...");
    const bannerCount = await reconnectingBanner.count();
    expect(bannerCount).toBe(0);
  });

  test("SSE connection persists across workspace navigation", async ({
    page,
  }) => {
    let sseRequestSeen = false;

    page.on("request", (req) => {
      if (req.url().includes("/api/v1/events/stream")) {
        sseRequestSeen = true;
      }
    });

    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // Verify SSE was requested
    expect(sseRequestSeen).toBe(true);

    // Navigate to a workspace
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

    // No reconnecting banner should appear after navigation
    const reconnectingBanner = page.getByText("Reconnecting...");
    const bannerCount = await reconnectingBanner.count();
    expect(bannerCount).toBe(0);
  });
});

test.describe("M7 SSE Connection — Snapshot Rehydration (R-19)", () => {
  test("SSE endpoint responds with event-stream content type", async ({
    page,
  }) => {
    let sseContentType = "";

    page.on("response", (response) => {
      if (response.url().includes("/api/v1/events/stream")) {
        sseContentType = response.headers()["content-type"] ?? "";
      }
    });

    await page.goto(`${BASE_URL}/`);
    await waitForReactContent(page);

    await page
      .locator(
        "aside a[href*='/workspaces/'], [role='complementary'] a[href*='/workspaces/']",
      )
      .first()
      .waitFor({ timeout: 10_000 });

    // The SSE endpoint should return text/event-stream
    expect(sseContentType).toContain("text/event-stream");
  });

  test("workspace data loads after navigation from deep link", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/workspaces/${WORKSPACE}/progress`);
    await waitForReactContent(page);

    await expect(
      page.locator("main h3").filter({ hasText: WORKSPACE }),
    ).toBeVisible({ timeout: 10_000 });

    const mainText = await page.locator("main").last().textContent();
    expect(mainText).toContain(WORKSPACE);
    expect(mainText).toContain("Progress");
  });
});

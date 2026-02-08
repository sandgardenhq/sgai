import { test, expect } from "@playwright/test";

const BASE_URL = "http://127.0.0.1:8181";
const WORKSPACE = "test-project";

// ─────────────────────────────────────────────────────────────────────────────
// M7 Headless Smoke Test — Embedded dist/ (R-22)
// Verifies the embedded `dist/` serves correctly from the Go binary.
// No sgai-ui cookie is set — React is the sole interface.
// ─────────────────────────────────────────────────────────────────────────────

test.describe("M7 Embedded dist/ Smoke Test (R-22)", () => {
  test("root serves React SPA with HTML shell", async ({ page }) => {
    const response = await page.goto(`${BASE_URL}/`);

    expect(response).not.toBeNull();
    expect(response!.status()).toBe(200);

    const contentType = response!.headers()["content-type"];
    expect(contentType).toContain("text/html");

    await page.locator("#root").waitFor({ timeout: 10_000 });
    await page.locator("#root > *").first().waitFor({ timeout: 10_000 });

    const title = await page.title();
    expect(title).toBeTruthy();
  });

  test("index.html contains script and stylesheet references", async ({
    page,
  }) => {
    await page.goto(`${BASE_URL}/`);

    const scripts = await page.locator("script[src]").count();
    expect(scripts).toBeGreaterThan(0);

    const stylesheets = await page.locator("link[rel='stylesheet']").count();
    expect(stylesheets).toBeGreaterThan(0);
  });

  test("CSS loads correctly with non-zero size", async ({ request }) => {
    const indexResponse = await request.get(`${BASE_URL}/`);
    const html = await indexResponse.text();

    const cssMatch = html.match(/href="(\/assets\/[^"]+\.css)"/);
    expect(cssMatch).not.toBeNull();

    const cssUrl = `${BASE_URL}${cssMatch![1]}`;
    const cssResponse = await request.get(cssUrl);
    expect(cssResponse.status()).toBe(200);

    const cssContentType = cssResponse.headers()["content-type"];
    expect(cssContentType).toContain("text/css");

    const cssBody = await cssResponse.text();
    expect(cssBody.length).toBeGreaterThan(100);
  });

  test("JS bundle loads correctly with non-zero size", async ({ request }) => {
    const indexResponse = await request.get(`${BASE_URL}/`);
    const html = await indexResponse.text();

    const jsMatch = html.match(/src="(\/assets\/[^"]+\.js)"/);
    expect(jsMatch).not.toBeNull();

    const jsUrl = `${BASE_URL}${jsMatch![1]}`;
    const jsResponse = await request.get(jsUrl);
    expect(jsResponse.status()).toBe(200);

    const jsContentType = jsResponse.headers()["content-type"];
    expect(
      jsContentType.includes("application/javascript") ||
        jsContentType.includes("text/javascript"),
    ).toBe(true);

    const jsBody = await jsResponse.text();
    expect(jsBody.length).toBeGreaterThan(100);
  });

  test("/trees redirects to root via React Router", async ({ page }) => {
    await page.goto(`${BASE_URL}/trees`);
    await page.locator("#root > *").first().waitFor({ timeout: 10_000 });

    expect(page.url()).toBe(`${BASE_URL}/`);
  });

  test("/api/v1/agents returns JSON, not index.html", async ({ request }) => {
    const response = await request.get(`${BASE_URL}/api/v1/agents`, {
      params: { workspace: WORKSPACE },
    });

    expect(response.status()).toBe(200);

    const contentType = response.headers()["content-type"];
    expect(contentType).toContain("application/json");

    const body = await response.json();
    expect(typeof body).toBe("object");
    expect(body).not.toBeNull();
    expect(Array.isArray(body.agents)).toBe(true);
    expect(body.agents.length).toBeGreaterThan(0);
  });

  test("/api/v1/workspaces returns JSON, not index.html", async ({
    request,
  }) => {
    const response = await request.get(`${BASE_URL}/api/v1/workspaces`);

    expect(response.status()).toBe(200);

    const contentType = response.headers()["content-type"];
    expect(contentType).toContain("application/json");

    const body = await response.json();
    expect(typeof body).toBe("object");
    expect(body).not.toBeNull();
    expect(Array.isArray(body.workspaces)).toBe(true);
  });

  test("/api/v1/skills returns JSON, not index.html", async ({ request }) => {
    const response = await request.get(`${BASE_URL}/api/v1/skills`, {
      params: { workspace: WORKSPACE },
    });

    expect(response.status()).toBe(200);

    const contentType = response.headers()["content-type"];
    expect(contentType).toContain("application/json");

    const body = await response.json();
    expect(Array.isArray(body) || typeof body === "object").toBe(true);
  });

  test("SPA catch-all serves index.html for unknown routes", async ({
    page,
  }) => {
    const response = await page.goto(`${BASE_URL}/some/unknown/route`);

    expect(response).not.toBeNull();
    expect(response!.status()).toBe(200);

    await page.locator("#root").waitFor({ timeout: 10_000 });
    await page.locator("#root > *").first().waitFor({ timeout: 10_000 });
  });

  test("static asset 404 for non-existent asset path", async ({ request }) => {
    const response = await request.get(
      `${BASE_URL}/assets/nonexistent-file.js`,
    );
    expect(response.status()).toBe(404);
  });

  test("no sgai-ui cookie needed — React loads by default", async ({
    page,
  }) => {
    const cookies = await page.context().cookies();
    const sgaiUiCookie = cookies.find((c) => c.name === "sgai-ui");
    expect(sgaiUiCookie).toBeUndefined();

    await page.goto(`${BASE_URL}/`);
    await page.locator("#root > *").first().waitFor({ timeout: 10_000 });

    await expect(page.getByText("SGAI").first()).toBeVisible();
  });
});

import { afterEach, describe, expect, it } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { StaleDiffRedirectRoute } from "../router-elements";

function LocationProbe() {
  const location = useLocation();
  return <output aria-label="current path">{location.pathname}</output>;
}

function renderStaleDiffRoute(initialEntry: string, routePath: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path={routePath} element={<StaleDiffRedirectRoute />} />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
});

describe("StaleDiffRedirectRoute", () => {
  it("redirects stale plural workspace diff deep links to progress", async () => {
    renderStaleDiffRoute("/workspaces/test-workspace/diff", "/workspaces/:name/diff");

    await waitFor(() => {
      expect(screen.getByLabelText("current path").textContent).toBe("/workspaces/test-workspace/progress");
    });
  });

  it("keeps redirecting stale singular workspace diff deep links to progress", async () => {
    renderStaleDiffRoute("/workspace/test-workspace/diff", "/workspace/:name/diff");

    await waitFor(() => {
      expect(screen.getByLabelText("current path").textContent).toBe("/workspaces/test-workspace/progress");
    });
  });
});

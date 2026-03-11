import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { resetFactoryStateStore, triggerFactoryRefresh } from "@/lib/factory-state";

const mockFetch = mock(() =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ workspaces: [] }),
  } as Response)
);

beforeEach(() => {
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  mockFetch.mockClear();
  mockFetch.mockImplementation(() =>
    Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ workspaces: [] }),
    } as Response)
  );
});

afterEach(() => {
  resetFactoryStateStore();
});

describe("factory-state store", () => {
  describe("resetFactoryStateStore", () => {
    it("resets the store without errors", () => {
      expect(() => resetFactoryStateStore()).not.toThrow();
    });

    it("can be called multiple times safely", () => {
      resetFactoryStateStore();
      resetFactoryStateStore();
      resetFactoryStateStore();
    });
  });

  describe("triggerFactoryRefresh", () => {
    it("triggers a refresh without errors", () => {
      expect(() => triggerFactoryRefresh()).not.toThrow();
    });

    it("calls fetch to /api/v1/state", async () => {
      triggerFactoryRefresh();
      await new Promise((resolve) => setTimeout(resolve, 400));
      const stateCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/state"
      );
      expect(stateCalls.length).toBeGreaterThan(0);
    });
  });

  describe("useFactoryState", () => {
    it("exports useFactoryState function", async () => {
      const mod = await import("@/lib/factory-state");
      expect(typeof mod.useFactoryState).toBe("function");
    });
  });

  describe("FetchStatus type", () => {
    it("exports FetchStatus type (verified by TypeScript compilation)", () => {
      const status: import("@/lib/factory-state").FetchStatus = "idle";
      expect(status).toBe("idle");
    });
  });

  describe("FactoryStateSnapshot type", () => {
    it("has correct shape", () => {
      const snapshot: import("@/lib/factory-state").FactoryStateSnapshot = {
        workspaces: [],
        fetchStatus: "idle",
        lastFetchedAt: null,
      };
      expect(snapshot.workspaces).toEqual([]);
      expect(snapshot.fetchStatus).toBe("idle");
      expect(snapshot.lastFetchedAt).toBeNull();
    });
  });

  describe("error handling", () => {
    it("handles fetch failure gracefully", async () => {
      mockFetch.mockImplementation(() => Promise.reject(new Error("Network error")));

      expect(() => triggerFactoryRefresh()).not.toThrow();
      await new Promise((resolve) => setTimeout(resolve, 400));
    });

    it("handles non-ok response gracefully", async () => {
      mockFetch.mockImplementation(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          json: () => Promise.reject(new Error("no json")),
        } as Response)
      );

      expect(() => triggerFactoryRefresh()).not.toThrow();
      await new Promise((resolve) => setTimeout(resolve, 400));
    });
  });
});

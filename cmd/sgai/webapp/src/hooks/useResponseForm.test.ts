import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { loadStoredState, saveStoredState, clearStoredState } from "./useResponseForm";

beforeEach(() => {
  sessionStorage.clear();
});

afterEach(() => {
  sessionStorage.clear();
});

describe("useResponseForm storage functions", () => {
  it("loadStoredState returns null when no data", () => {
    const result = loadStoredState("prefix-", "workspace");
    expect(result).toBeNull();
  });

  it("saveStoredState and loadStoredState round-trip", () => {
    const state = {
      selections: { "0": ["Option A"] },
      otherText: "hello",
      questionId: "q-123",
    };

    saveStoredState("prefix-", "workspace", state);
    const loaded = loadStoredState("prefix-", "workspace");

    expect(loaded).not.toBeNull();
    expect(loaded!.selections).toEqual({ "0": ["Option A"] });
    expect(loaded!.otherText).toBe("hello");
    expect(loaded!.questionId).toBe("q-123");
  });

  it("clearStoredState removes data", () => {
    saveStoredState("prefix-", "workspace", {
      selections: {},
      otherText: "data",
      questionId: "q-456",
    });

    clearStoredState("prefix-", "workspace");
    const loaded = loadStoredState("prefix-", "workspace");
    expect(loaded).toBeNull();
  });

  it("uses prefix to namespace storage keys", () => {
    saveStoredState("modal-", "ws1", {
      selections: {},
      otherText: "modal data",
      questionId: "q-1",
    });

    saveStoredState("page-", "ws1", {
      selections: {},
      otherText: "page data",
      questionId: "q-2",
    });

    const modalLoaded = loadStoredState("modal-", "ws1");
    const pageLoaded = loadStoredState("page-", "ws1");

    expect(modalLoaded!.otherText).toBe("modal data");
    expect(pageLoaded!.otherText).toBe("page data");
  });

  it("loadStoredState handles invalid JSON gracefully", () => {
    sessionStorage.setItem("prefix-workspace", "not-json");
    const result = loadStoredState("prefix-", "workspace");
    expect(result).toBeNull();
  });
});

import { describe, it, expect, beforeEach, afterEach, afterAll, mock, spyOn } from "bun:test";
import { renderHook } from "@testing-library/react";
import { useNotifications } from "./useNotifications";
import { api } from "../lib/api";
import type { ApiWorkspacesResponse } from "../types";

import {
  useSSEEvent as _realUseSSEEvent,
  useSSEStore as _realUseSSEStore,
  useWorkspaceSSEEvent as _realUseWorkspaceSSEEvent,
  useConnectionStatus as _realUseConnectionStatus,
} from "./useSSE";

const savedExports = {
  useSSEEvent: _realUseSSEEvent,
  useSSEStore: _realUseSSEStore,
  useWorkspaceSSEEvent: _realUseWorkspaceSSEEvent,
  useConnectionStatus: _realUseConnectionStatus,
};

let mockSSEEvent: unknown = null;

mock.module("./useSSE", () => ({
  useSSEEvent: () => mockSSEEvent,
}));

afterAll(() => {
  mock.module("./useSSE", () => savedExports);
});

class MockNotification {
  static permission: NotificationPermission = "granted";
  static requestPermission = mock(() => Promise.resolve("granted" as NotificationPermission));
  static instances: MockNotification[] = [];

  title: string;
  options: NotificationOptions;
  onclick: ((ev: Event) => void) | null = null;

  constructor(title: string, options: NotificationOptions = {}) {
    this.title = title;
    this.options = options;
    MockNotification.instances.push(this);
  }
}

const OriginalNotification = globalThis.Notification;

function setNotificationAPI(permission: NotificationPermission): void {
  MockNotification.permission = permission;
  MockNotification.requestPermission = mock(() => Promise.resolve(permission));
  MockNotification.instances = [];
  Object.defineProperty(globalThis, "Notification", {
    value: MockNotification,
    writable: true,
    configurable: true,
  });
}

function restoreNotificationAPI(): void {
  if (OriginalNotification) {
    Object.defineProperty(globalThis, "Notification", {
      value: OriginalNotification,
      writable: true,
      configurable: true,
    });
  }
}

function makeWorkspacesResponse(
  workspaces: { name: string; needsInput: boolean; forks?: { name: string; needsInput: boolean }[] }[],
): ApiWorkspacesResponse {
  return {
    workspaces: workspaces.map((ws) => ({
      name: ws.name,
      dir: `/tmp/${ws.name}`,
      running: false,
      needsInput: ws.needsInput,
      inProgress: false,
      pinned: false,
      isRoot: true,
      status: "idle",
      hasSgai: true,
      forks: ws.forks?.map((f) => ({
        name: f.name,
        dir: `/tmp/${f.name}`,
        running: false,
        needsInput: f.needsInput,
        inProgress: false,
        pinned: false,
        isRoot: false,
        status: "idle",
        hasSgai: true,
      })),
    })),
  };
}

function flushPromises(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe("useNotifications", () => {
  let listSpy: ReturnType<typeof spyOn>;

  beforeEach(() => {
    mockSSEEvent = null;
    setNotificationAPI("granted");
    listSpy = spyOn(api.workspaces, "list");
  });

  afterEach(() => {
    restoreNotificationAPI();
    listSpy.mockRestore();
  });

  it("does not fetch when no SSE event received", () => {
    mockSSEEvent = null;
    renderHook(() => useNotifications());
    expect(listSpy).not.toHaveBeenCalled();
  });

  it("fetches workspaces on SSE event", async () => {
    const response = makeWorkspacesResponse([
      { name: "project-a", needsInput: false },
    ]);
    listSpy.mockResolvedValue(response);

    mockSSEEvent = { ts: 1 };
    renderHook(() => useNotifications());

    await flushPromises();
    expect(listSpy).toHaveBeenCalledTimes(1);
  });

  it("fires notification on false-to-true transition", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );

    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(0);

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].title).toBe("Approval Needed");
    expect(MockNotification.instances[0].options.body).toBe(
      "Workspace project-a needs your input",
    );
    expect(MockNotification.instances[0].options.tag).toBe("project-a");
  });

  it("does not fire notification when needsInput stays false", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );

    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("does not re-fire notification when needsInput stays true", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );

    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    // First appearance with needsInput=true fires once (unknown→true transition)
    expect(MockNotification.instances).toHaveLength(1);

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    // Second event with same state should NOT fire again
    expect(MockNotification.instances).toHaveLength(1);
  });

  it("re-fires notification after needsInput cycles back through false to true", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();
    expect(MockNotification.instances).toHaveLength(1);

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 3 };
    rerender();
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 4 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(2);
  });

  it("fires notification for nested forks", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([
        {
          name: "root-ws",
          needsInput: false,
          forks: [{ name: "fork-1", needsInput: false }],
        },
      ]),
    );

    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([
        {
          name: "root-ws",
          needsInput: false,
          forks: [{ name: "fork-1", needsInput: true }],
        },
      ]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].options.body).toBe(
      "Workspace fork-1 needs your input",
    );
  });

  it("does not fire notification when permission is default", async () => {
    setNotificationAPI("default");

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("skips notification when permission is denied", async () => {
    setNotificationAPI("denied");

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("handles API errors gracefully", async () => {
    listSpy.mockRejectedValueOnce(new Error("Network error"));

    mockSSEEvent = { ts: 1 };
    renderHook(() => useNotifications());
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("sets onclick handler on notification", async () => {
    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: false }]),
    );
    mockSSEEvent = { ts: 1 };
    const { rerender } = renderHook(() => useNotifications());
    await flushPromises();

    listSpy.mockResolvedValueOnce(
      makeWorkspacesResponse([{ name: "project-a", needsInput: true }]),
    );
    mockSSEEvent = { ts: 2 };
    rerender();
    await flushPromises();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].onclick).toBeFunction();
  });
});

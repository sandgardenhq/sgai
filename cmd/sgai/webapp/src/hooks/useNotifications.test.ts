import { describe, it, expect, beforeEach, afterEach, afterAll, mock } from "bun:test";
import { renderHook, act } from "@testing-library/react";
import { useNotifications } from "./useNotifications";
import type { FactoryStateSnapshot } from "../lib/factory-state";

import { useFactoryState as _realUseFactoryState } from "../lib/factory-state";

const savedExports = {
  useFactoryState: _realUseFactoryState,
};

let mockFactoryState: FactoryStateSnapshot = {
  workspaces: [],
  fetchStatus: "idle",
  lastFetchedAt: null,
};

mock.module("../lib/factory-state", () => ({
  useFactoryState: () => mockFactoryState,
  resetFactoryStateStore: () => {},
}));

afterAll(() => {
  mock.module("../lib/factory-state", () => savedExports);
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

function makeWorkspaces(
  workspaces: { name: string; needsInput: boolean; forks?: { name: string; needsInput: boolean }[] }[],
): FactoryStateSnapshot["workspaces"] {
  return workspaces.map((ws) => ({
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
  }));
}

describe("useNotifications", () => {
  beforeEach(() => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: null,
    };
    setNotificationAPI("granted");
  });

  afterEach(() => {
    restoreNotificationAPI();
  });

  it("does not fire notification when state not yet fetched", () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: null,
    };
    renderHook(() => useNotifications());
    expect(MockNotification.instances).toHaveLength(0);
  });

  it("does not fire notification when needsInput stays false", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    expect(MockNotification.instances).toHaveLength(0);

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("fires notification on false-to-true transition", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    expect(MockNotification.instances).toHaveLength(0);

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].title).toBe("Approval Needed");
    expect(MockNotification.instances[0].options.body).toBe(
      "Workspace project-a needs your input",
    );
    expect(MockNotification.instances[0].options.tag).toBe("project-a");
  });

  it("does not re-fire notification when needsInput stays true", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    expect(MockNotification.instances).toHaveLength(1);

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(1);
  });

  it("re-fires notification after needsInput cycles back through false to true", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();
    expect(MockNotification.instances).toHaveLength(1);

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 2,
      };
    });
    rerender();

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 3,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(2);
  });

  it("fires notification for nested forks", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([
        {
          name: "root-ws",
          needsInput: false,
          forks: [{ name: "fork-1", needsInput: false }],
        },
      ]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([
          {
            name: "root-ws",
            needsInput: false,
            forks: [{ name: "fork-1", needsInput: true }],
          },
        ]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].options.body).toBe(
      "Workspace fork-1 needs your input",
    );
  });

  it("does not fire notification when permission is default", () => {
    setNotificationAPI("default");

    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("skips notification when permission is denied", () => {
    setNotificationAPI("denied");

    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(0);
  });

  it("sets onclick handler on notification", () => {
    mockFactoryState = {
      workspaces: makeWorkspaces([{ name: "project-a", needsInput: false }]),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };

    const { rerender } = renderHook(() => useNotifications());

    act(() => {
      mockFactoryState = {
        workspaces: makeWorkspaces([{ name: "project-a", needsInput: true }]),
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender();

    expect(MockNotification.instances).toHaveLength(1);
    expect(MockNotification.instances[0].onclick).toBeFunction();
  });
});

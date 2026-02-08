import { describe, test, expect, beforeEach, afterEach, mock } from "bun:test";
import { createSSEStore } from "./sse-store";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map();
  readyState: number = 0;
  closed = false;

  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;

  constructor(url: string) {
    this.url = url;
    this.readyState = MockEventSource.CONNECTING;
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    existing.push(listener);
    this.listeners.set(type, existing);
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    this.listeners.set(
      type,
      existing.filter((l) => l !== listener),
    );
  }

  close() {
    this.closed = true;
    this.readyState = MockEventSource.CLOSED;
  }

  simulateOpen() {
    this.readyState = MockEventSource.OPEN;
    this.onopen?.();
  }

  simulateError() {
    this.onerror?.();
  }

  simulateEvent(type: string, data: string) {
    const event = new MessageEvent(type, { data });
    const listeners = this.listeners.get(type) ?? [];
    for (const listener of listeners) {
      listener(event);
    }
  }
}

let mockEventSources: MockEventSource[] = [];
const OriginalEventSource = globalThis.EventSource;

beforeEach(() => {
  mockEventSources = [];
  (globalThis as unknown as { EventSource: typeof MockEventSource }).EventSource =
    class extends MockEventSource {
      constructor(url: string) {
        super(url);
        mockEventSources.push(this);
      }
    } as unknown as typeof EventSource;
});

afterEach(() => {
  (globalThis as unknown as { EventSource: typeof EventSource }).EventSource =
    OriginalEventSource;
});

describe("createSSEStore", () => {
  test("starts with disconnected status", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const snapshot = store.getSnapshot();
    expect(snapshot.connectionStatus).toBe("disconnected");
    expect(snapshot.lastEvent).toBeNull();
    store.destroy();
  });

  test("connects on first subscribe", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});
    expect(mockEventSources.length).toBe(1);
    expect(mockEventSources[0].url).toBe("/api/v1/events/stream");
    unsubscribe();
    store.destroy();
  });

  test("updates status to connected on open", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const updates: string[] = [];

    const unsubscribe = store.subscribe(() => {
      updates.push(store.getSnapshot().connectionStatus);
    });

    mockEventSources[0].simulateOpen();

    expect(store.getSnapshot().connectionStatus).toBe("connected");
    unsubscribe();
    store.destroy();
  });

  test("handles typed SSE events", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});

    mockEventSources[0].simulateOpen();

    const testData = { workspaces: [{ name: "test" }] };
    mockEventSources[0].simulateEvent(
      "workspace:update",
      JSON.stringify(testData),
    );

    const snapshot = store.getSnapshot();
    expect(snapshot.lastEvent?.type).toBe("workspace:update");
    expect(snapshot.events["workspace:update"]).toEqual(testData);

    unsubscribe();
    store.destroy();
  });

  test("handles non-JSON event data", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});

    mockEventSources[0].simulateOpen();
    mockEventSources[0].simulateEvent("log:append", "plain text log line");

    const snapshot = store.getSnapshot();
    expect(snapshot.events["log:append"]).toBe("plain text log line");

    unsubscribe();
    store.destroy();
  });

  test("disconnects and schedules reconnect on error", async () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});

    mockEventSources[0].simulateOpen();
    expect(store.getSnapshot().connectionStatus).toBe("connected");

    mockEventSources[0].simulateError();
    expect(store.getSnapshot().connectionStatus).toBe("disconnected");
    expect(mockEventSources[0].closed).toBe(true);

    unsubscribe();
    store.destroy();
  });

  test("exponential backoff increases retry delay", async () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});

    mockEventSources[0].simulateOpen();
    mockEventSources[0].simulateError();

    await new Promise((r) => setTimeout(r, 1100));
    expect(mockEventSources.length).toBe(2);

    mockEventSources[1].simulateError();

    await new Promise((r) => setTimeout(r, 2100));
    expect(mockEventSources.length).toBe(3);

    unsubscribe();
    store.destroy();
  });

  test("resets retry delay on successful connection", async () => {
    const store = createSSEStore("/api/v1/events/stream");
    const unsubscribe = store.subscribe(() => {});

    mockEventSources[0].simulateOpen();
    mockEventSources[0].simulateError();

    await new Promise((r) => setTimeout(r, 1100));

    mockEventSources[1].simulateOpen();
    mockEventSources[1].simulateError();

    await new Promise((r) => setTimeout(r, 1100));
    expect(mockEventSources.length).toBe(3);

    unsubscribe();
    store.destroy();
  });

  test("closes connection when all listeners unsubscribe", () => {
    const store = createSSEStore("/api/v1/events/stream");

    const unsub1 = store.subscribe(() => {});
    const unsub2 = store.subscribe(() => {});

    expect(mockEventSources[0].closed).toBe(false);

    unsub1();
    expect(mockEventSources[0].closed).toBe(false);

    unsub2();
    expect(mockEventSources[0].closed).toBe(true);

    store.destroy();
  });

  test("destroy cleans up completely", () => {
    const store = createSSEStore("/api/v1/events/stream");
    store.subscribe(() => {});

    mockEventSources[0].simulateOpen();
    expect(store.getSnapshot().connectionStatus).toBe("connected");

    store.destroy();
    expect(mockEventSources[0].closed).toBe(true);
    expect(store.getSnapshot().connectionStatus).toBe("disconnected");
  });

  test("getServerSnapshot returns disconnected state", () => {
    const store = createSSEStore("/api/v1/events/stream");
    const serverSnapshot = store.getServerSnapshot();
    expect(serverSnapshot.connectionStatus).toBe("disconnected");
    expect(serverSnapshot.lastEvent).toBeNull();
    store.destroy();
  });

  test("notifies all listeners on event", () => {
    const store = createSSEStore("/api/v1/events/stream");
    let count1 = 0;
    let count2 = 0;

    const unsub1 = store.subscribe(() => { count1++; });
    const unsub2 = store.subscribe(() => { count2++; });

    mockEventSources[0].simulateOpen();

    expect(count1).toBeGreaterThan(0);
    expect(count2).toBeGreaterThan(0);

    const prev1 = count1;
    const prev2 = count2;

    mockEventSources[0].simulateEvent(
      "session:update",
      JSON.stringify({ status: "running" }),
    );

    expect(count1).toBeGreaterThan(prev1);
    expect(count2).toBeGreaterThan(prev2);

    unsub1();
    unsub2();
    store.destroy();
  });
});

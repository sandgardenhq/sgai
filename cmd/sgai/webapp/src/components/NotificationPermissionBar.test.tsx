import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import { NotificationPermissionBar } from "./NotificationPermissionBar";

class MockNotification {
  static permission: NotificationPermission = "default";
  static requestPermission = mock(() =>
    Promise.resolve("granted" as NotificationPermission),
  );
}

const OriginalNotification = globalThis.Notification;

function setNotificationAPI(permission: NotificationPermission): void {
  MockNotification.permission = permission;
  MockNotification.requestPermission = mock(() =>
    Promise.resolve(permission === "default" ? "granted" : permission),
  );
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

describe("NotificationPermissionBar", () => {
  beforeEach(() => {
    localStorage.clear();
    setNotificationAPI("default");
  });

  afterEach(() => {
    restoreNotificationAPI();
    cleanup();
  });

  it("shows bar when permission is default and not dismissed", () => {
    render(<NotificationPermissionBar />);
    expect(
      screen.getByText(/enable browser notifications/i),
    ).toBeTruthy();
  });

  it("does not show bar when permission is granted", () => {
    setNotificationAPI("granted");
    render(<NotificationPermissionBar />);
    expect(
      screen.queryByText(/enable browser notifications/i),
    ).toBeNull();
  });

  it("does not show bar when permission is denied", () => {
    setNotificationAPI("denied");
    render(<NotificationPermissionBar />);
    expect(
      screen.queryByText(/enable browser notifications/i),
    ).toBeNull();
  });

  it("does not show bar when previously dismissed", () => {
    localStorage.setItem("notification-permission-dismissed", "true");
    render(<NotificationPermissionBar />);
    expect(
      screen.queryByText(/enable browser notifications/i),
    ).toBeNull();
  });

  it("hides bar and stores dismissal when dismiss is clicked", () => {
    render(<NotificationPermissionBar />);
    const dismissButton = screen.getByText("Dismiss");
    fireEvent.click(dismissButton);

    expect(
      screen.queryByText(/enable browser notifications/i),
    ).toBeNull();
    expect(localStorage.getItem("notification-permission-dismissed")).toBe(
      "true",
    );
  });

  it("calls requestPermission when enable is clicked", async () => {
    render(<NotificationPermissionBar />);
    const enableButton = screen.getByText("Enable");
    fireEvent.click(enableButton);

    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(MockNotification.requestPermission).toHaveBeenCalledTimes(1);
  });

  it("does not show bar when Notification API is not available", () => {
    const saved = globalThis.Notification;
    // biome-ignore lint/performance/noDelete: test needs to fully remove the property
    delete (globalThis as Record<string, unknown>).Notification;
    try {
      render(<NotificationPermissionBar />);
      expect(
        screen.queryByText(/enable browser notifications/i),
      ).toBeNull();
    } finally {
      Object.defineProperty(globalThis, "Notification", {
        value: saved,
        writable: true,
        configurable: true,
      });
    }
  });
});

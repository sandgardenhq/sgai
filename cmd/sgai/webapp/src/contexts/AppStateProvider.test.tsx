import { describe, test, expect } from "bun:test";
import { renderHook, act } from "@testing-library/react";
import {
  AppStateProvider,
  useAppState,
  useAppDispatch,
} from "./AppStateProvider";
import type { ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }) {
  return <AppStateProvider>{children}</AppStateProvider>;
}

describe("AppStateProvider", () => {
  test("provides initial state", () => {
    const { result } = renderHook(() => useAppState(), { wrapper });

    expect(result.current.selectedWorkspace).toBeNull();
    expect(result.current.ui.panelCollapsed).toBe(false);
    expect(result.current.ui.activeTab).toBe("goal");
  });

  test("workspace/select action updates selectedWorkspace", () => {
    const { result } = renderHook(
      () => ({
        state: useAppState(),
        dispatch: useAppDispatch(),
      }),
      { wrapper },
    );

    act(() => {
      result.current.dispatch({
        type: "workspace/select",
        workspace: "my-project",
      });
    });

    expect(result.current.state.selectedWorkspace).toBe("my-project");
  });

  test("ui/togglePanel toggles panel collapsed state", () => {
    const { result } = renderHook(
      () => ({
        state: useAppState(),
        dispatch: useAppDispatch(),
      }),
      { wrapper },
    );

    expect(result.current.state.ui.panelCollapsed).toBe(false);

    act(() => {
      result.current.dispatch({ type: "ui/togglePanel" });
    });

    expect(result.current.state.ui.panelCollapsed).toBe(true);

    act(() => {
      result.current.dispatch({ type: "ui/togglePanel" });
    });

    expect(result.current.state.ui.panelCollapsed).toBe(false);
  });

  test("ui/setTab updates active tab", () => {
    const { result } = renderHook(
      () => ({
        state: useAppState(),
        dispatch: useAppDispatch(),
      }),
      { wrapper },
    );

    act(() => {
      result.current.dispatch({ type: "ui/setTab", tab: "session" });
    });

    expect(result.current.state.ui.activeTab).toBe("session");
  });

  test("useAppState throws when used outside provider", () => {
    expect(() => {
      renderHook(() => useAppState());
    }).toThrow("useAppState must be used within an AppStateProvider");
  });

  test("useAppDispatch throws when used outside provider", () => {
    expect(() => {
      renderHook(() => useAppDispatch());
    }).toThrow("useAppDispatch must be used within an AppStateProvider");
  });
});

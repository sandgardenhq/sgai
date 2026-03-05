import { describe, it, expect, afterEach } from "bun:test";
import { renderHook, cleanup, act } from "@testing-library/react";
import { useIsMobile } from "@/hooks/use-mobile";

afterEach(() => {
  cleanup();
});

describe("useIsMobile", () => {
  it("returns false for desktop-width windows", () => {
    Object.defineProperty(window, "innerWidth", { value: 1024, writable: true });
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);
  });

  it("returns true for mobile-width windows", () => {
    Object.defineProperty(window, "innerWidth", { value: 375, writable: true });
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(true);
  });

  it("returns false at exactly the breakpoint", () => {
    Object.defineProperty(window, "innerWidth", { value: 768, writable: true });
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);
  });

  it("returns true below the breakpoint", () => {
    Object.defineProperty(window, "innerWidth", { value: 767, writable: true });
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(true);
  });
});

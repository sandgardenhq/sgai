import { describe, it, expect, beforeEach } from "bun:test";
import { renderHook, act } from "@testing-library/react";
import { useSidebarResize } from "@/hooks/useSidebarResize";

beforeEach(() => {
  localStorage.clear();
});

describe("useSidebarResize", () => {
  describe("initial state", () => {
    it("returns default width of 256 when no stored value", () => {
      const { result } = renderHook(() => useSidebarResize());
      expect(result.current.sidebarWidth).toBe(256);
    });

    it("returns stored width from localStorage", () => {
      localStorage.setItem("sidebar-width", "300");
      const { result } = renderHook(() => useSidebarResize());
      expect(result.current.sidebarWidth).toBe(300);
    });

    it("clamps stored width to minimum of 192", () => {
      localStorage.setItem("sidebar-width", "100");
      const { result } = renderHook(() => useSidebarResize());
      expect(result.current.sidebarWidth).toBe(192);
    });

    it("clamps stored width to maximum of 384", () => {
      localStorage.setItem("sidebar-width", "500");
      const { result } = renderHook(() => useSidebarResize());
      expect(result.current.sidebarWidth).toBe(384);
    });

    it("uses default when stored value is not a number", () => {
      localStorage.setItem("sidebar-width", "invalid");
      const { result } = renderHook(() => useSidebarResize());
      expect(result.current.sidebarWidth).toBe(256);
    });
  });

  describe("handleMouseDown", () => {
    it("returns a function", () => {
      const { result } = renderHook(() => useSidebarResize());
      expect(typeof result.current.handleMouseDown).toBe("function");
    });
  });

  describe("resize interaction", () => {
    it("updates width on mouse move after mouse down", () => {
      const { result } = renderHook(() => useSidebarResize());

      act(() => {
        result.current.handleMouseDown({
          preventDefault: () => {},
          clientX: 256,
        } as React.MouseEvent);
      });

      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 300 }));
      });

      expect(result.current.sidebarWidth).toBe(300);
    });

    it("persists width to localStorage on mouse up", () => {
      const { result } = renderHook(() => useSidebarResize());

      act(() => {
        result.current.handleMouseDown({
          preventDefault: () => {},
          clientX: 256,
        } as React.MouseEvent);
      });

      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 300 }));
      });

      act(() => {
        window.dispatchEvent(new MouseEvent("mouseup"));
      });

      expect(localStorage.getItem("sidebar-width")).toBe("300");
    });

    it("clamps width during drag to minimum", () => {
      const { result } = renderHook(() => useSidebarResize());

      act(() => {
        result.current.handleMouseDown({
          preventDefault: () => {},
          clientX: 256,
        } as React.MouseEvent);
      });

      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 50 }));
      });

      expect(result.current.sidebarWidth).toBe(192);
    });

    it("clamps width during drag to maximum", () => {
      const { result } = renderHook(() => useSidebarResize());

      act(() => {
        result.current.handleMouseDown({
          preventDefault: () => {},
          clientX: 256,
        } as React.MouseEvent);
      });

      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 700 }));
      });

      expect(result.current.sidebarWidth).toBe(384);
    });

    it("does not update width on mouse move without mouse down", () => {
      const { result } = renderHook(() => useSidebarResize());

      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 400 }));
      });

      expect(result.current.sidebarWidth).toBe(256);
    });
  });
});

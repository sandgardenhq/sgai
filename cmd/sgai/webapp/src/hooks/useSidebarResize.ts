import { useState, useEffect, useEffectEvent, useRef, useCallback } from "react";

const STORAGE_KEY = "sidebar-width";
const MIN_WIDTH = 192;
const MAX_WIDTH = 384;
const DEFAULT_WIDTH = 256;

function clampWidth(width: number): number {
  return Math.min(Math.max(width, MIN_WIDTH), MAX_WIDTH);
}

function readStoredWidth(): number {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = Number(stored);
      if (!Number.isNaN(parsed)) {
        return clampWidth(parsed);
      }
    }
  } catch {
  }
  return DEFAULT_WIDTH;
}

function setBodyResizeStyle(enabled: boolean): void {
  const style = document.body.style;
  if (enabled) {
    style.cssText = `${style.cssText}; cursor: col-resize; user-select: none;`;
    return;
  }
  style.cssText = style.cssText
    .replace(/(?:^|;)\s*cursor\s*:[^;]*/g, "")
    .replace(/(?:^|;)\s*user-select\s*:[^;]*/g, "");
}

export interface SidebarResizeResult {
  sidebarWidth: number;
  handleMouseDown: (e: React.MouseEvent) => void;
}

export function useSidebarResize(): SidebarResizeResult {
  const [sidebarWidth, setSidebarWidth] = useState<number>(readStoredWidth);
  const isDragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(0);

  const moveSidebarResize = useEffectEvent((e: MouseEvent) => {
    if (!isDragging.current) return;
    const delta = e.clientX - startX.current;
    const newWidth = clampWidth(startWidth.current + delta);
    setSidebarWidth(newWidth);
  });

  const stopSidebarResize = useEffectEvent(() => {
    if (!isDragging.current) return;
    isDragging.current = false;
    setBodyResizeStyle(false);
    setSidebarWidth((w) => {
      try {
        localStorage.setItem(STORAGE_KEY, String(w));
      } catch {
      }
      return w;
    });
  });

  useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => moveSidebarResize(event);
    const handleMouseUp = () => stopSidebarResize();
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, []);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    startX.current = e.clientX;
    startWidth.current = sidebarWidth;
    setBodyResizeStyle(true);
  }, [sidebarWidth]);

  return { sidebarWidth, handleMouseDown };
}

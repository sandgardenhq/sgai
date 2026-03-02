import { useState, useEffect, useRef, useCallback } from "react";

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

export interface SidebarResizeResult {
  sidebarWidth: number;
  handleMouseDown: (e: React.MouseEvent) => void;
}

export function useSidebarResize(): SidebarResizeResult {
  const [sidebarWidth, setSidebarWidth] = useState<number>(readStoredWidth);
  const isDragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(0);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDragging.current) return;
    const delta = e.clientX - startX.current;
    const newWidth = clampWidth(startWidth.current + delta);
    setSidebarWidth(newWidth);
  }, []);

  const handleMouseUp = useCallback(() => {
    if (!isDragging.current) return;
    isDragging.current = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    setSidebarWidth((w) => {
      try {
        localStorage.setItem(STORAGE_KEY, String(w));
      } catch {
      }
      return w;
    });
  }, []);

  useEffect(() => {
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [handleMouseMove, handleMouseUp]);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    startX.current = e.clientX;
    startWidth.current = sidebarWidth;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }, [sidebarWidth]);

  return { sidebarWidth, handleMouseDown };
}

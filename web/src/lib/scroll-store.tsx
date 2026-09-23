"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { SectionId } from "./site";

type ScrollState = {
  progress: number;
  active: SectionId;
  setActive: (id: SectionId) => void;
};

const ScrollCtx = createContext<ScrollState | null>(null);

export function ScrollProvider({ children }: { children: ReactNode }) {
  const [progress, setProgress] = useState(0);
  const [active, setActive] = useState<SectionId>("hook");

  useEffect(() => {
    let raf = 0;
    const onScroll = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => {
        const max = Math.max(
          1,
          document.documentElement.scrollHeight - window.innerHeight,
        );
        setProgress(Math.min(1, Math.max(0, window.scrollY / max)));
      });
    };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
    };
  }, []);

  const setActiveStable = useCallback((id: SectionId) => {
    setActive((prev) => (prev === id ? prev : id));
  }, []);

  const value = useMemo(
    () => ({ progress, active, setActive: setActiveStable }),
    [progress, active, setActiveStable],
  );

  return <ScrollCtx.Provider value={value}>{children}</ScrollCtx.Provider>;
}

export function useScrollState() {
  const ctx = useContext(ScrollCtx);
  if (!ctx) throw new Error("useScrollState outside ScrollProvider");
  return ctx;
}

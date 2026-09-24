"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { SectionId } from "./site";
import { SECTIONS } from "./site";

const FLASH_MS = 2000;

type FlashState = { id: SectionId; token: number } | null;

type ScrollState = {
  progress: number;
  active: SectionId;
  setActive: (id: SectionId) => void;
  flash: FlashState;
  /** Smooth-scroll to a section and highlight it for 2s. */
  goToSection: (id: SectionId) => void;
};

const ScrollCtx = createContext<ScrollState | null>(null);

type LenisLike = {
  scrollTo: (
    target: string | number | HTMLElement,
    opts?: { offset?: number; immediate?: boolean },
  ) => void;
};

const LenisCtx = createContext<LenisLike | null>(null);

export function useLenis() {
  return useContext(LenisCtx);
}

export function LenisProvider({
  lenis,
  children,
}: {
  lenis: LenisLike | null;
  children: ReactNode;
}) {
  return <LenisCtx.Provider value={lenis}>{children}</LenisCtx.Provider>;
}

export function isSectionId(value: string): value is SectionId {
  return SECTIONS.some((s) => s.id === value);
}

export function ScrollProvider({ children }: { children: ReactNode }) {
  const [progress, setProgress] = useState(0);
  const [active, setActive] = useState<SectionId>("hook");
  const [flash, setFlash] = useState<FlashState>(null);
  const flashTimer = useRef(0);
  const lenis = useLenis();

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

  useEffect(() => {
    return () => window.clearTimeout(flashTimer.current);
  }, []);

  const setActiveStable = useCallback((id: SectionId) => {
    setActive((prev) => (prev === id ? prev : id));
  }, []);

  const goToSection = useCallback(
    (id: SectionId) => {
      const el = document.getElementById(id);
      if (!el) return;

      if (lenis) {
        lenis.scrollTo(el, { offset: 0 });
      } else {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }

      window.history.replaceState(null, "", `#${id}`);
      setActiveStable(id);

      window.clearTimeout(flashTimer.current);
      setFlash({ id, token: Date.now() });
      flashTimer.current = window.setTimeout(() => {
        setFlash((prev) => (prev?.id === id ? null : prev));
      }, FLASH_MS);
    },
    [lenis, setActiveStable],
  );

  const value = useMemo(
    () => ({
      progress,
      active,
      setActive: setActiveStable,
      flash,
      goToSection,
    }),
    [progress, active, setActiveStable, flash, goToSection],
  );

  return <ScrollCtx.Provider value={value}>{children}</ScrollCtx.Provider>;
}

export function useScrollState() {
  const ctx = useContext(ScrollCtx);
  if (!ctx) throw new Error("useScrollState outside ScrollProvider");
  return ctx;
}

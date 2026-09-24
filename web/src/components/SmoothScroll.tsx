"use client";

import Lenis from "lenis";
import { useEffect, useState, type ReactNode } from "react";
import { LenisProvider } from "@/lib/scroll-store";

export function SmoothScroll({ children }: { children: ReactNode }) {
  const [lenis, setLenis] = useState<Lenis | null>(null);

  useEffect(() => {
    const instance = new Lenis({
      duration: 1.15,
      smoothWheel: true,
      touchMultiplier: 1.1,
    });
    setLenis(instance);

    let raf = 0;
    const tick = (time: number) => {
      instance.raf(time);
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(raf);
      instance.destroy();
      setLenis(null);
    };
  }, []);

  return <LenisProvider lenis={lenis}>{children}</LenisProvider>;
}

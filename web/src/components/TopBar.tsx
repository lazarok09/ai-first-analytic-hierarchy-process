"use client";

import { SITE } from "@/lib/site";
import { useScrollState } from "@/lib/scroll-store";

export function TopBar() {
  const { active } = useScrollState();

  return (
    <header className="fixed inset-x-0 top-0 z-50 flex items-center justify-between gap-4 px-[clamp(1rem,4vw,2.5rem)] py-4 mix-blend-normal">
      <a href="#hook" className="font-display text-lg font-bold tracking-tight text-steel-bright">
        {SITE.name}
        <span className="ml-2 font-mono text-[0.65rem] font-normal tracking-[0.2em] text-amber uppercase">
          cli
        </span>
      </a>
      <nav className="flex items-center gap-2 sm:gap-3" aria-label="Product">
        <span className="hidden font-mono text-[0.65rem] tracking-[0.18em] text-steel-dim uppercase md:inline">
          {active}
        </span>
        <a className="cta" href={SITE.github} target="_blank" rel="noreferrer">
          GitHub
        </a>
        <a className="cta cta-primary" href="#close">
          Install
        </a>
      </nav>
    </header>
  );
}

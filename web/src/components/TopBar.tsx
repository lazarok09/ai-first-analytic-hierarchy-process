"use client";

import { SECTIONS, SITE } from "@/lib/site";
import { useScrollState } from "@/lib/scroll-store";

export function TopBar() {
  const { active, goToSection } = useScrollState();

  return (
    <header className="fixed inset-x-0 top-0 z-50 px-[clamp(1rem,2.5vw,1.75rem)] py-4 mix-blend-normal">
      <div className="stage flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <a
            href="#hook"
            className="font-display text-lg font-bold tracking-tight text-steel-bright"
            onClick={(e) => {
              e.preventDefault();
              goToSection("hook");
            }}
          >
            {SITE.name}
            <span className="ml-2 font-mono text-[0.65rem] font-normal tracking-[0.2em] text-amber uppercase">
              cli
            </span>
          </a>

          <nav className="flex items-center gap-2 sm:gap-3" aria-label="Product">
            <a className="cta" href={SITE.github} target="_blank" rel="noreferrer">
              GitHub
            </a>
            <a
              className="cta cta-primary"
              href="#close"
              onClick={(e) => {
                e.preventDefault();
                goToSection("close");
              }}
            >
              Install
            </a>
          </nav>
        </div>

        <nav className="toc" aria-label="Table of contents">
          {SECTIONS.map((section) => {
            const isActive = active === section.id;
            return (
              <a
                key={section.id}
                href={`#${section.id}`}
                title={section.line}
                aria-current={isActive ? "true" : undefined}
                className={`toc-link ${isActive ? "toc-link-active" : ""}`}
                onClick={(e) => {
                  e.preventDefault();
                  goToSection(section.id);
                }}
              >
                <span className="toc-num" aria-hidden>
                  {section.kicker.slice(0, 2)}
                </span>
                <span className="toc-title">{section.title}</span>
              </a>
            );
          })}
        </nav>
      </div>
    </header>
  );
}

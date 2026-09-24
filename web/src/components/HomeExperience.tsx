"use client";

import dynamic from "next/dynamic";
import { InstallCommand } from "@/components/InstallCommand";
import { ManifestoSection } from "@/components/ManifestoSection";
import { SmoothScroll } from "@/components/SmoothScroll";
import { TopBar } from "@/components/TopBar";
import { ScrollProvider, useScrollState } from "@/lib/scroll-store";
import { SECTIONS, SITE } from "@/lib/site";
import type { ReactNode } from "react";

const Scene = dynamic(
  () => import("@/components/Scene").then((m) => m.Scene),
  { ssr: false },
);

function SectionLink({
  id,
  className,
  children,
}: {
  id: (typeof SECTIONS)[number]["id"];
  className?: string;
  children: ReactNode;
}) {
  const { goToSection } = useScrollState();
  return (
    <a
      className={className}
      href={`#${id}`}
      onClick={(e) => {
        e.preventDefault();
        goToSection(id);
      }}
    >
      {children}
    </a>
  );
}

export function HomeExperience() {
  const [hook, lie, stance, agent, proof, close] = SECTIONS;

  return (
    <SmoothScroll>
      <ScrollProvider>
        <Scene />
        <TopBar />

        <main>
          <ManifestoSection {...hook}>
            <div className="flex flex-wrap gap-3">
              <a className="cta cta-primary" href={SITE.github} target="_blank" rel="noreferrer">
                Open source
              </a>
              <SectionLink className="cta" id="close">
                Install
              </SectionLink>
            </div>
          </ManifestoSection>

          <ManifestoSection {...lie} align="right" />
          <ManifestoSection {...stance} />
          <ManifestoSection {...agent} align="right" />
          <ManifestoSection {...proof} />

          <ManifestoSection {...close}>
            <div className="flex w-full max-w-xl flex-col gap-4">
              <InstallCommand />
              <div className="flex flex-wrap gap-3">
                <a className="cta cta-primary" href={SITE.github} target="_blank" rel="noreferrer">
                  GitHub
                </a>
                <a className="cta" href={SITE.releases} target="_blank" rel="noreferrer">
                  Releases
                </a>
                <a className="cta" href={SITE.license} target="_blank" rel="noreferrer">
                  MIT License
                </a>
              </div>
            </div>
          </ManifestoSection>
        </main>

        <footer className="relative z-10 border-t border-white/5 px-[clamp(1.25rem,2.5vw,1.75rem)] py-8 font-mono text-xs tracking-wide text-steel-dim">
          <div className="stage flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <p>
              {SITE.name} — pairwise judgment. Eigenvector priorities. Human commit.
            </p>
            <p>
              <a className="hover:text-amber" href={SITE.github} target="_blank" rel="noreferrer">
                source
              </a>
              <span className="mx-2 opacity-40">/</span>
              <a className="hover:text-amber" href={SITE.releases} target="_blank" rel="noreferrer">
                releases
              </a>
            </p>
          </div>
        </footer>
      </ScrollProvider>
    </SmoothScroll>
  );
}
